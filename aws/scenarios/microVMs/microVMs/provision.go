package microVMs

import (
	"fmt"
	"os"
	"path/filepath"

	sconfig "github.com/DataDog/test-infra-definitions/aws/scenarios/microVMs/config"
	"github.com/DataDog/test-infra-definitions/aws/scenarios/microVMs/vmconfig"
	"github.com/DataDog/test-infra-definitions/command"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	basefsName           = "custom-fsbase"
	libvirtSSHPrivateKey = "libvirt_rsa"
	libvirtSSHPublicKey  = "libvirt_rsa.pub"
)

var (
	downloadKernelArgs = command.CommandArgs{
		Create: pulumi.String("wget -q https://dd-agent-omnibus.s3.amazonaws.com/kernel-version-testing/kernel-packages.tar.gz -O /tmp/kernel-packages.tar.gz"),
	}
	disableSELinuxArgs = command.CommandArgs{
		Create: pulumi.String("sed --in-place 's/#security_driver = \"selinux\"/security_driver = \"none\"/' /etc/libvirt/qemu.conf"),
		Sudo:   true,
	}
	libvirtReadyArgs = command.CommandArgs{
		Create: pulumi.String("systemctl restart libvirtd"),
		Sudo:   true,
	}
	poolDefineReadyArgs = command.CommandArgs{
		Create: pulumi.String("virsh pool-define /tmp/pool.xml"),
		Sudo:   true,
	}
	buildSharedDirArgs = command.CommandArgs{
		Create: pulumi.String("install -d -m 0777 -o libvirt-qemu -g kvm /opt/kernel-headers"),
		Sudo:   true,
	}
)

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

func downloadAndExtractKernelPackage(runner *command.Runner) ([]pulumi.Resource, error) {
	downloadKernelPackage, err := runner.Command("download-kernel-image", &downloadKernelArgs)
	if err != nil {
		return []pulumi.Resource{}, err
	}

	extractPackageArgs := command.CommandArgs{
		Create: pulumi.String("pushd /tmp; tar -xzvf kernel-packages.tar.gz; popd;"),
	}
	extractPackageDone, err := runner.Command("extract-kernel-packages", &extractPackageArgs, pulumi.DependsOn([]pulumi.Resource{downloadKernelPackage}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	return []pulumi.Resource{extractPackageDone}, nil
}

func copyKernelHeaders(runner *command.Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	permissionFixArgs := command.CommandArgs{
		Create: pulumi.String("chown -R libvirt-qemu:kvm /tmp/kernel-packages"),
		Sudo:   true,
	}
	permissionFixDone, err := runner.Command("permission-fix-kernel-headers", &permissionFixArgs, pulumi.DependsOn(depends))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	copyKernelHeadersArgs := command.CommandArgs{
		Create: pulumi.String("-u libvirt-qemu /bin/bash -c \"cd /tmp; find /tmp/kernel-packages -name 'linux-image-*' -type f | xargs -i cp {} /opt/kernel-headers && find /tmp/kernel-packages -name 'linux-headers-*' -type f | xargs -i cp {} /opt/kernel-headers\""),
		Sudo:   true,
	}
	copyKernelHeadersDone, err := runner.Command("copy-kernel-headers", &copyKernelHeadersArgs, pulumi.DependsOn([]pulumi.Resource{permissionFixDone}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	return []pulumi.Resource{copyKernelHeadersDone}, nil
}

func installPackages(runner *command.Runner) ([]pulumi.Resource, error) {
	aptManager := command.NewAptManager(runner)
	installSocat, err := aptManager.Ensure("socat")
	if err != nil {
		return []pulumi.Resource{}, err
	}

	installQemu, err := aptManager.Ensure("qemu-kvm", pulumi.DependsOn([]pulumi.Resource{installSocat}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	installLibVirt, err := aptManager.Ensure("libvirt-daemon-system", pulumi.DependsOn([]pulumi.Resource{installQemu}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	return []pulumi.Resource{installLibVirt}, nil
}

func prepareLibvirtEnvironment(runner *command.Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	disableSELinux, err := runner.Command("disable-selinux-qemu", &disableSELinuxArgs, pulumi.DependsOn(depends))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	libvirtReady, err := runner.Command("restart-libvirtd", &libvirtReadyArgs, pulumi.DependsOn([]pulumi.Resource{disableSELinux}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	return []pulumi.Resource{libvirtReady}, nil
}

func prepareLibvirtSSHKeys(runner *command.Runner, localRunner *command.LocalRunner, depends []pulumi.Resource, tempDir string) ([]pulumi.Resource, error) {
	privateKeyPath := filepath.Join(tempDir, libvirtSSHPrivateKey)
	publicKeyPath := filepath.Join(tempDir, libvirtSSHPublicKey)
	sshGenArgs := command.CommandArgs{
		Create: pulumi.String(
			fmt.Sprintf("rm -f %s && rm -f %s && ssh-keygen -t rsa -b 4096 -f %s -q -N \"\" && cat %s", privateKeyPath, publicKeyPath, privateKeyPath, publicKeyPath),
		),
		Update: pulumi.String("true"),
		Delete: pulumi.String(fmt.Sprintf("rm %s && rm %s", privateKeyPath, publicKeyPath)),
	}
	sshgenDone, err := localRunner.Command("gen-libvirt-sshkey", &sshGenArgs)
	if err != nil {
		return []pulumi.Resource{}, err
	}

	sshWriteArgs := command.CommandArgs{
		Create: pulumi.Sprintf("echo '%s' >> ~/.ssh/authorized_keys", sshgenDone.Stdout),
		Update: pulumi.String("true"),
		Sudo:   true,
	}
	sshWrite, err := runner.Command("write-ssh-key", &sshWriteArgs, pulumi.DependsOn([]pulumi.Resource{sshgenDone}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	return []pulumi.Resource{sshWrite}, nil
}

// This function provisions the metal instance for setting up libvirt based micro-vms.
func provisionInstance(runner *command.Runner, localRunner *command.LocalRunner, m *sconfig.DDMicroVMConfig) ([]pulumi.Resource, error) {
	packagesInstallDone, err := installPackages(runner)
	if err != nil {
		return []pulumi.Resource{}, err
	}

	prepareLibvirtEnvDone, err := prepareLibvirtEnvironment(runner, packagesInstallDone)
	if err != nil {
		return []pulumi.Resource{}, err
	}

	downloadKernelDone, err := downloadAndExtractKernelPackage(runner)
	if err != nil {
		return []pulumi.Resource{}, err
	}

	// We need to wait until the libvirt-qemu user exists before doing this
	// Hence, the dependency on the libvirt environment.
	buildSharedDirDone, err := runner.Command("build-kernel-headers-dir", &buildSharedDirArgs, pulumi.DependsOn(prepareLibvirtEnvDone))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	dependencies := append(downloadKernelDone, buildSharedDirDone)
	copyKernelHeadersDone, err := copyKernelHeaders(runner, dependencies)
	if err != nil {
		return []pulumi.Resource{}, err
	}

	tempDir := m.GetStringWithDefault(m.MicroVMConfig, "tempDir", "/tmp")
	prepareSSHKeysDone, err := prepareLibvirtSSHKeys(runner, localRunner, []pulumi.Resource{}, tempDir)
	if err != nil {
		return []pulumi.Resource{}, err
	}

	return append(prepareSSHKeysDone, copyKernelHeadersDone...), nil

}

func downloadRootfs(vmset vmconfig.VMSet, runner *command.Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	downloadRootfsArgs := command.CommandArgs{
		Create: pulumi.String(
			fmt.Sprintf("wget -q %s -O /tmp/bullseye.qcow2.amd64-0.1-DEV.tar.gz", vmset.Img.ImageSourceURI),
		),
	}

	res, err := runner.Command("download-rootfs", &downloadRootfsArgs, pulumi.DependsOn(depends))
	return []pulumi.Resource{res}, err
}

func extractRootfs(vmset vmconfig.VMSet, runner *command.Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	extractTopLevelArchive := command.CommandArgs{
		Create: pulumi.String(
			fmt.Sprintf("pushd /tmp; tar -xzf bullseye.qcow2.amd64-0.1-DEV.tar.gz; popd;"),
		),
	}
	res, err := runner.Command("extract-base-volume-package", &extractTopLevelArchive, pulumi.DependsOn(depends))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	return []pulumi.Resource{res}, err
}

func setupLibvirtVMSetPool(vmset vmconfig.VMSet, runner *command.Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	poolBuildReadyArgs := command.CommandArgs{
		Create: pulumi.String(
			fmt.Sprintf("virsh pool-build %s", vmset.Name),
		),
		Sudo: true,
	}
	poolStartReadyArgs := command.CommandArgs{
		Create: pulumi.String(
			fmt.Sprintf("virsh pool-start %s", vmset.Name),
		),
		Sudo: true,
	}
	poolRefreshDoneArgs := command.CommandArgs{
		Create: pulumi.String(
			fmt.Sprintf("virsh pool-refresh %s", vmset.Name),
		),
		Sudo: true,
	}

	poolDefineReady, err := runner.Command("define-libvirt-pool", &poolDefineReadyArgs, pulumi.DependsOn(depends))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	poolBuildReady, err := runner.Command("build-libvirt-pool", &poolBuildReadyArgs, pulumi.DependsOn([]pulumi.Resource{poolDefineReady}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	poolStartReady, err := runner.Command("start-libvirt-pool", &poolStartReadyArgs, pulumi.DependsOn([]pulumi.Resource{poolBuildReady}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	poolRefreshDone, err := runner.Command("refresh-libvirt-pool", &poolRefreshDoneArgs, pulumi.DependsOn([]pulumi.Resource{poolStartReady}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	return []pulumi.Resource{poolRefreshDone}, err
}

func setupLibvirtVMVolume(vmset vmconfig.VMSet, runner *command.Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	baseVolumeReadyArgs := command.CommandArgs{
		Create: pulumi.String(fmt.Sprintf("virsh vol-create %s /tmp/volume.xml", vmset.Name)),
		Sudo:   true,
	}
	uploadImageToVolumeReadyArgs := command.CommandArgs{
		Create: pulumi.String(
			fmt.Sprintf("virsh vol-upload %s %s --pool %s", generateVolumeKey(vmset.Name), vmset.Img.ImagePath, vmset.Name),
		),
		Sudo: true,
	}

	baseVolumeReady, err := runner.Command("build-libvirt-basevolume", &baseVolumeReadyArgs, pulumi.DependsOn(depends))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	uploadImageToVolumeReady, err := runner.Command("upload-libvirt-volume", &uploadImageToVolumeReadyArgs, pulumi.DependsOn([]pulumi.Resource{baseVolumeReady}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	return []pulumi.Resource{uploadImageToVolumeReady}, err
}

func setupLibvirtFilesystem(vmset vmconfig.VMSet, runner *command.Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	downloadRootfsDone, err := downloadRootfs(vmset, runner, []pulumi.Resource{})
	if err != nil {
		return []pulumi.Resource{}, err
	}

	extractRootfsDone, err := extractRootfs(vmset, runner, downloadRootfsDone)
	if err != nil {
		return []pulumi.Resource{}, err
	}

	poolXml, err := generatePoolXML(vmset.Name)
	if err != nil {
		return []pulumi.Resource{}, err
	}
	poolXmlWrittenArgs := command.CommandArgs{
		Create: pulumi.String(
			fmt.Sprintf("echo \"%s\" > /tmp/pool.xml", string(poolXml)),
		),
	}
	poolXmlWritten, err := runner.Command("write-pool-xml", &poolXmlWrittenArgs, pulumi.DependsOn(depends))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	volXml, err := generateVolumeXML(vmset.Name)
	if err != nil {
		return []pulumi.Resource{}, err
	}
	volXmlWrittenArgs := command.CommandArgs{
		Create: pulumi.String(
			fmt.Sprintf("echo \"%s\" > /tmp/volume.xml", string(volXml)),
		),
	}
	volXmlWritten, err := runner.Command("write-vol-xml", &volXmlWrittenArgs, pulumi.DependsOn(depends))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	setupLibvirtVMPoolDone, err := setupLibvirtVMSetPool(vmset, runner, []pulumi.Resource{poolXmlWritten})
	if err != nil {
		return []pulumi.Resource{}, err
	}

	setupLibvirtVMVolumeDone, err := setupLibvirtVMVolume(vmset, runner, append(
		append([]pulumi.Resource{volXmlWritten}, extractRootfsDone...), setupLibvirtVMPoolDone...),
	)
	if err != nil {
		return []pulumi.Resource{}, err
	}

	return setupLibvirtVMVolumeDone, nil
}
