package microvms

import (
	"fmt"
	"path/filepath"

	"github.com/DataDog/test-infra-definitions/aws/ec2"
	sconfig "github.com/DataDog/test-infra-definitions/aws/scenarios/microVMs/config"
	"github.com/DataDog/test-infra-definitions/command"
	"github.com/DataDog/test-infra-definitions/common/namer"
	"github.com/DataDog/test-infra-definitions/common/os"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	libvirtSSHPrivateKeyX86 = "libvirt_rsa-x86"
	libvirtSSHPrivateKeyArm = "libvirt_rsa-arm"
	sharedFSMountPoint      = "/opt/kernel-version-testing"
)

var SSHKeyFileNames = map[string]string{
	ec2.AMD64Arch: libvirtSSHPrivateKeyX86,
	ec2.ARM64Arch: libvirtSSHPrivateKeyArm,
}

var kernelHeadersDir = filepath.Join(sharedFSMountPoint, "kernel-headers")

var (
	disableSELinuxArgs = command.Args{
		Create: pulumi.String("sed --in-place 's/#security_driver = \"selinux\"/security_driver = \"none\"/' /etc/libvirt/qemu.conf"),
		Sudo:   true,
	}
	libvirtSockPerms = command.Args{
		Create: pulumi.String("sed --in-place 's/#unix_sock_group = \"libvirt\"/unix_sock_group = \"libvirt\"/g' /etc/libvirt/libvirtd.conf && sed --in-place 's/#unix_sock_ro_perms = \"0777\"/unix_sock_ro_perms = \"0777\"/g' /etc/libvirt/libvirtd.conf && sed --in-place 's/#unix_sock_rw_perms = \"0770\"/unix_sock_rw_perms = \"0770\"/g' /etc/libvirt/libvirtd.conf "),
		Sudo:   true,
	}
	libvirtReadyArgs = command.Args{
		Create: pulumi.String("systemctl restart libvirtd"),
		Sudo:   true,
	}

	buildSharedDirArgs = command.Args{
		Create: pulumi.Sprintf("install -d -m 0777 -o $USER -g $USER %s", sharedFSMountPoint),
		Delete: pulumi.Sprintf("rm -rf %s", sharedFSMountPoint),
		Sudo:   true,
	}

	buildKernelHeadersDirArgs = command.Args{
		Create: pulumi.Sprintf("install -d -m 0777 -o $USER -g $USER %s", kernelHeadersDir),
		Delete: pulumi.Sprintf("rm -rf %s", kernelHeadersDir),
		Sudo:   true,
	}
)

var GetWorkingDirectory func() string

func getKernelVersionTestingWorkingDir(m *sconfig.DDMicroVMConfig) func() string {
	return func() string {
		return m.GetStringWithDefault(m.MicroVMConfig, sconfig.DDMicroVMWorkingDirectory, "/tmp")
	}
}

func downloadAndExtractKernelPackage(runner *Runner, arch string, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	kernelPackages := fmt.Sprintf("kernel-packages-%s.tar", arch)
	kernelPackagesDownloadDir := filepath.Join(GetWorkingDirectory(), "kernel-packages")

	kernelPackagesDownloadTarget := filepath.Join(kernelPackagesDownloadDir, kernelPackages)
	downloadKernelArgs := command.Args{
		Create: pulumi.Sprintf("wget -q https://dd-agent-omnibus.s3.amazonaws.com/kernel-version-testing/%s -O %s", kernelPackages, kernelPackagesDownloadTarget),
	}
	downloadKernelPackage, err := runner.Command("download-kernel-image", &downloadKernelArgs, pulumi.DependsOn(depends))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	extractPackageArgs := command.Args{
		Create: pulumi.Sprintf("cd %s && tar xvf %s | xargs -i tar xzf {};", kernelPackagesDownloadDir, kernelPackages),
	}
	extractPackageDone, err := runner.Command("extract-kernel-packages", &extractPackageArgs, pulumi.DependsOn([]pulumi.Resource{downloadKernelPackage}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	return []pulumi.Resource{extractPackageDone}, nil
}

func copyKernelHeaders(runner *Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	kernelPackagesDownloadDir := filepath.Join(GetWorkingDirectory(), "kernel-packages")

	copyKernelHeadersArgs := command.Args{
		Create: pulumi.Sprintf(
			"cd %s && find %s -name 'linux-image-*' -type f | xargs -i cp {} %s && find %s -name 'linux-headers-*' -type f | xargs -i cp {} %s", GetWorkingDirectory(), kernelPackagesDownloadDir, kernelHeadersDir, kernelPackagesDownloadDir, kernelHeadersDir,
		),
	}
	copyKernelHeadersDone, err := runner.Command("copy-kernel-headers", &copyKernelHeadersArgs, pulumi.DependsOn(depends))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	return []pulumi.Resource{copyKernelHeadersDone}, nil
}

func installPackages(runner *Runner) ([]pulumi.Resource, error) {
	remoteRunner, err := runner.GetRemoteRunner()
	if err != nil {
		return []pulumi.Resource{}, fmt.Errorf("failed to install packages: %w", err)
	}
	aptManager := os.NewAptManager(remoteRunner)
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

func prepareLibvirtEnvironment(runner *Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	disableSELinux, err := runner.Command("disable-selinux-qemu", &disableSELinuxArgs, pulumi.DependsOn(depends))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	setLibvirtSockPerms, err := runner.Command("libvirt-sock-perms", &libvirtSockPerms, pulumi.DependsOn(depends))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	libvirtReady, err := runner.Command("restart-libvirtd", &libvirtReadyArgs, pulumi.DependsOn([]pulumi.Resource{disableSELinux, setLibvirtSockPerms}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	return []pulumi.Resource{libvirtReady}, nil
}

func prepareLibvirtSSHKeys(runner *Runner, localRunner *command.LocalRunner, resourceNamer namer.Namer, pair sshKeyPair, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	sshGenArgs := command.Args{
		Create: pulumi.Sprintf("rm -f %s && rm -f %s && ssh-keygen -t rsa -b 4096 -f %s -q -N \"\" && cat %s", pair.privateKey, pair.publicKey, pair.privateKey, pair.publicKey),
		Delete: pulumi.Sprintf("rm %s && rm %s", pair.privateKey, pair.publicKey),
	}
	sshgenDone, err := localRunner.Command(resourceNamer.ResourceName("gen-libvirt-sshkey"), &sshGenArgs)
	if err != nil {
		return []pulumi.Resource{}, err
	}

	// This command writes the public ssh key which pulumi uses to talk to the libvirt daemon, in the authorized_keys
	// file of the default user. We must write in this file because pulumi runs its commands as the default user.
	//
	// We override the runner-level user here with root, and construct the path to the default users .ssh directory,
	// in order to write the public ssh key in the correct file.
	sshWriteArgs := command.Args{
		Create: pulumi.Sprintf("echo '%s' >> $(getent passwd 1000 | cut -d: -f6)/.ssh/authorized_keys", sshgenDone.Stdout),
		Sudo:   true,
	}

	wait := append(depends, sshgenDone)
	sshWrite, err := runner.Command("write-ssh-key", &sshWriteArgs, pulumi.DependsOn(wait))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	return []pulumi.Resource{sshWrite}, nil
}

func buildDirectoryStructure(runner *Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	kernelPackagesDir := filepath.Join(GetWorkingDirectory(), "kernel-packages")
	rootfsDir := filepath.Join(GetWorkingDirectory(), "rootfs")

	buildDirectoryStructureArgs := command.Args{
		Create: pulumi.Sprintf("install -d -m 0755 -o $USER -g kvm %s && install -d -m 0755 -o $USER -g kvm %s", kernelPackagesDir, rootfsDir),
		Sudo:   true,
	}
	buildDirectoryStructureDone, err := runner.Command("build-directory-structure", &buildDirectoryStructureArgs, pulumi.DependsOn(depends))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	return []pulumi.Resource{buildDirectoryStructureDone}, nil
}

// This function provisions the metal instance for setting up libvirt based micro-vms.
func provisionInstance(instance *Instance) ([]pulumi.Resource, error) {
	runner := instance.runner

	packagesInstallDone, err := installPackages(runner)
	if err != nil {
		return []pulumi.Resource{}, err
	}

	prepareLibvirtEnvDone, err := prepareLibvirtEnvironment(runner, packagesInstallDone)
	if err != nil {
		return []pulumi.Resource{}, err
	}

	buildDirectoryStructureDone, err := buildDirectoryStructure(runner, prepareLibvirtEnvDone)
	if err != nil {
		return []pulumi.Resource{}, err
	}

	// We need to wait until the libvirt-qemu user exists before doing this
	// Hence, the dependency on the libvirt environment.
	buildSharedDirDone, err := runner.Command("build-kernel-version-testing-dir", &buildSharedDirArgs, pulumi.DependsOn(prepareLibvirtEnvDone))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	buildKernelHeadersDirDone, err := runner.Command("build-kernel-headers-dir", &buildKernelHeadersDirArgs, pulumi.DependsOn([]pulumi.Resource{buildSharedDirDone}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	kernelPackagesDone, err := setupKernelPackages(instance, append(buildDirectoryStructureDone, buildKernelHeadersDirDone))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	return kernelPackagesDone, nil
}

func setupKernelPackages(instance *Instance, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	downloadKernelDone, err := downloadAndExtractKernelPackage(instance.runner, instance.Arch, depends)
	if err != nil {
		return []pulumi.Resource{}, err
	}

	copyKernelHeadersDone, err := copyKernelHeaders(instance.runner, downloadKernelDone)
	if err != nil {
		return []pulumi.Resource{}, err
	}

	return copyKernelHeadersDone, nil
}
