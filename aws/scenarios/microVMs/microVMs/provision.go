package microVMs

import (
	"fmt"
	"path/filepath"

	sconfig "github.com/DataDog/test-infra-definitions/aws/scenarios/microVMs/config"
	"github.com/DataDog/test-infra-definitions/command"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	libvirtSSHPrivateKey = "libvirt_rsa"
	libvirtSSHPublicKey  = "libvirt_rsa.pub"
	sharedFSMountPoint   = "/opt/kernel-version-testing"
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

	buildSharedDirArgs = command.CommandArgs{
		Create: pulumi.String(
			fmt.Sprintf("install -d -m 0777 -o libvirt-qemu -g kvm %s", sharedFSMountPoint),
		),
		Delete: pulumi.String(
			fmt.Sprintf("rm -rf %s", sharedFSMountPoint),
		),
		Sudo: true,
	}
)

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
		Create: pulumi.String(
			fmt.Sprintf("-u libvirt-qemu /bin/bash -c \"cd /tmp; find /tmp/kernel-packages -name 'linux-image-*' -type f | xargs -i cp {} %s && find /tmp/kernel-packages -name 'linux-headers-*' -type f | xargs -i cp {} %s\"", sharedFSMountPoint, sharedFSMountPoint),
		),
		Sudo: true,
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
		Delete: pulumi.String(fmt.Sprintf("rm %s && rm %s", privateKeyPath, publicKeyPath)),
	}
	sshgenDone, err := localRunner.Command("gen-libvirt-sshkey", &sshGenArgs)
	if err != nil {
		return []pulumi.Resource{}, err
	}

	sshWriteArgs := command.CommandArgs{
		Create: pulumi.Sprintf("echo '%s' >> ~/.ssh/authorized_keys", sshgenDone.Stdout),
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
