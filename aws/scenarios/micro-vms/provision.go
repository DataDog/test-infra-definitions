package main

import (
	"fmt"
	"os"

	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/aws/scenarios/micro-vms/vmconfig"
	"github.com/DataDog/test-infra-definitions/command"
	awsEc2 "github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const basefsName = "custom-fsbase"

var LibvirtPrivateKey string

var (
	downloadKernelArgs  = command.CommandArgs{Create: pulumi.String("wget -q https://dd-agent-omnibus.s3.amazonaws.com/bzImage -O /tmp/bzImage")}
	disableSELinuxArgs  = command.CommandArgs{Create: pulumi.String("sed --in-place 's/#security_driver = \"selinux\"/security_driver = \"none\"/' /etc/libvirt/qemu.conf"), Sudo: true}
	libvirtReadyArgs    = command.CommandArgs{Create: pulumi.String("systemctl restart libvirtd"), Sudo: true}
	downloadRootfsArgs  = command.CommandArgs{Create: pulumi.String("wget -q https://dd-agent-omnibus.s3.amazonaws.com/rootfs.tar.gz -O /tmp/rootfs.tar.gz")}
	poolDefineReadyArgs = command.CommandArgs{Create: pulumi.String("virsh pool-define /tmp/pool.xml"), Sudo: true}
)

func createRunner(vm *awsEc2.Instance, conn remote.ConnectionOutput, e aws.Environment) (*command.Runner, error) {
	runner, err := command.NewRunner(*e.CommonEnvironment, e.Ctx.Stack()+"-conn", conn, func(r *command.Runner) (*remote.Command, error) {
		return command.WaitForCloudInit(e.Ctx, r)
	})
	if err != nil {
		return nil, err
	}

	return runner, nil
}

func generatePoolPath(name string) string {
	return "/pool/" + name + "/"
}

func generateVolumeKey(pool string) string {
	return generatePoolPath(pool) + basefsName
}

func generateVolumeXML(pool string) (string, error) {
	xml, err := os.ReadFile("resources/volume.xml")
	if err != nil {
		return "", err
	}
	volumeXml := string(xml)
	key := generateVolumeKey(pool)
	path := key

	return fmt.Sprintf(volumeXml, basefsName, key, path), nil
}

func generatePoolXML(pool string) (string, error) {
	xml, err := os.ReadFile("resources/pool.xml")
	if err != nil {
		return "", err
	}

	path := generatePoolPath(pool)
	return fmt.Sprintf(string(xml), pool, path), nil
}

func provisionInstance(runner *command.Runner) ([]pulumi.Resource, error) {
	downloadKernel, err := runner.Command("download-kernel-image", &downloadKernelArgs)
	aptManager := command.NewAptManager(runner)

	installQemu, err := aptManager.Ensure("qemu-kvm")
	if err != nil {
		return []pulumi.Resource{}, err
	}

	installLibVirt, err := aptManager.Ensure("libvirt-daemon-system", pulumi.DependsOn([]pulumi.Resource{installQemu}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	disableSELinux, err := runner.Command("disable-selinux-qemu", &disableSELinuxArgs, pulumi.DependsOn([]pulumi.Resource{installLibVirt}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	libvirtReady, err := runner.Command("restart-libvirtd", &libvirtReadyArgs, pulumi.DependsOn([]pulumi.Resource{disableSELinux}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	privKey, pubKey, err := generateSSHKeyPair()
	if err != nil {
		return []pulumi.Resource{}, err
	}

	LibvirtPrivateKey, err = writeKeyToTempFile(privKey, "libvirt_rsa")
	if err != nil {
		return []pulumi.Resource{}, err
	}

	sshWriteArgs := command.CommandArgs{Create: pulumi.String(fmt.Sprintf("sudo echo \"%s\" >> ~/.ssh/authorized_keys", string(pubKey)))}
	sshWrite, err := runner.Command("write-ssh-key", &sshWriteArgs)
	if err != nil {
		return []pulumi.Resource{}, err
	}

	return []pulumi.Resource{libvirtReady, sshWrite, downloadKernel}, nil

}

func setupLibvirtFilesystem(vmset vmconfig.VMSet, runner *command.Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	downloadRootfs, err := runner.Command("download-rootfs", &downloadRootfsArgs)

	extractRootfsArgs := command.CommandArgs{Create: pulumi.String(fmt.Sprintf("tar xzOf /tmp/rootfs.tar.gz > %s", vmset.Image))}
	extractRootfs, err := runner.Command("extract-rootfs", &extractRootfsArgs, pulumi.DependsOn([]pulumi.Resource{downloadRootfs}))

	poolXml, err := generatePoolXML(vmset.Name)
	if err != nil {
		return []pulumi.Resource{}, err
	}

	poolXmlWrittenArgs := command.CommandArgs{Create: pulumi.String(fmt.Sprintf("echo \"%s\" > /tmp/pool.xml", string(poolXml)))}
	poolXmlWritten, err := runner.Command("write-pool-xml", &poolXmlWrittenArgs, pulumi.DependsOn(depends))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	poolDefineReady, err := runner.Command("define-libvirt-pool", &poolDefineReadyArgs, pulumi.DependsOn([]pulumi.Resource{poolXmlWritten}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	poolBuildReadyArgs := command.CommandArgs{Create: pulumi.String(fmt.Sprintf("virsh pool-build %s", vmset.Name)), Sudo: true}
	poolBuildReady, err := runner.Command("build-libvirt-pool", &poolBuildReadyArgs, pulumi.DependsOn([]pulumi.Resource{poolDefineReady}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	poolStartReadyArgs := command.CommandArgs{Create: pulumi.String(fmt.Sprintf("virsh pool-start %s", vmset.Name)), Sudo: true}
	poolStartReady, err := runner.Command("start-libvirt-pool", &poolStartReadyArgs, pulumi.DependsOn([]pulumi.Resource{poolBuildReady}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	poolRefreshDoneArgs := command.CommandArgs{Create: pulumi.String(fmt.Sprintf("virsh pool-refresh %s", vmset.Name)), Sudo: true}
	poolRefreshDone, err := runner.Command("refresh-libvirt-pool", &poolRefreshDoneArgs, pulumi.DependsOn([]pulumi.Resource{poolStartReady}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	volXml, err := generateVolumeXML(vmset.Name)
	if err != nil {
		return []pulumi.Resource{}, err
	}

	volXmlWrittenArgs := command.CommandArgs{Create: pulumi.String(fmt.Sprintf("echo \"%s\" > /tmp/volume.xml", string(volXml)))}
	volXmlWritten, err := runner.Command("write-vol-xml", &volXmlWrittenArgs, pulumi.DependsOn(depends))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	baseVolumeReadyArgs := command.CommandArgs{Create: pulumi.String(fmt.Sprintf("virsh vol-create %s /tmp/volume.xml", vmset.Name)), Sudo: true}
	baseVolumeReady, err := runner.Command("build-libvirt-basevolume", &baseVolumeReadyArgs, pulumi.DependsOn([]pulumi.Resource{poolRefreshDone, extractRootfs, volXmlWritten}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	uploadImageToVolumeReadyArgs := command.CommandArgs{Create: pulumi.String(fmt.Sprintf("virsh vol-upload %s %s --pool %s", generateVolumeKey(vmset.Name), vmset.Image, vmset.Name)), Sudo: true}
	uploadImageToVolumeReady, err := runner.Command("upload-libvirt-volume", &uploadImageToVolumeReadyArgs, pulumi.DependsOn([]pulumi.Resource{baseVolumeReady}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	return []pulumi.Resource{uploadImageToVolumeReady}, nil
}
