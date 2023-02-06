package microVM

import (
	"fmt"
	"path/filepath"

	sconfig "github.com/DataDog/test-infra-definitions/aws/scenarios/microVMs/config"
	"github.com/DataDog/test-infra-definitions/command"
	"github.com/DataDog/test-infra-definitions/common/namer"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	libvirtSSHPrivateKey = "libvirt_rsa-%s"
	libvirtSSHPublicKey  = "libvirt_rsa-%s.pub"
	sharedFSMountPoint   = "/opt/kernel-version-testing"
)

var kernelHeadersDir = filepath.Join(sharedFSMountPoint, "kernel-headers")

var (
	disableSELinuxArgs = command.Args{
		Create: pulumi.String("sed --in-place 's/#security_driver = \"selinux\"/security_driver = \"none\"/' /etc/libvirt/qemu.conf"),
		Sudo:   true,
	}
	libvirtReadyArgs = command.Args{
		Create: pulumi.String("systemctl restart libvirtd"),
		Sudo:   true,
	}

	buildSharedDirArgs = command.Args{
		Create: pulumi.Sprintf("install -d -m 0777 -o libvirt-qemu -g kvm %s", sharedFSMountPoint),
		Delete: pulumi.Sprintf("rm -rf %s", sharedFSMountPoint),
		Sudo:   true,
	}

	buildKernelHeadersDirArgs = command.Args{
		Create: pulumi.Sprintf("install -d -m 0777 -o libvirt-qemu -g kvm %s", kernelHeadersDir),
		Delete: pulumi.Sprintf("rm -rf %s", kernelHeadersDir),
		Sudo:   true,
	}
)

func downloadAndExtractKernelPackage(runner *command.Runner, arch string) ([]pulumi.Resource, error) {
	kernelPackages := fmt.Sprintf("kernel-packages-%s.tar", arch)
	downloadKernelArgs := command.Args{
		Create: pulumi.Sprintf("mkdir /tmp/kernel-packages && wget -q https://dd-agent-omnibus.s3.amazonaws.com/kernel-version-testing/%s -O /tmp/kernel-packages/%s", kernelPackages, kernelPackages),
	}
	downloadKernelPackage, err := runner.Command("download-kernel-image", &downloadKernelArgs)
	if err != nil {
		return []pulumi.Resource{}, err
	}

	extractPackageArgs := command.Args{
		Create: pulumi.Sprintf("pushd /tmp/kernel-packages; tar xvf %s | xargs -i tar xzf {}; popd;", kernelPackages),
	}
	extractPackageDone, err := runner.Command("extract-kernel-packages", &extractPackageArgs, pulumi.DependsOn([]pulumi.Resource{downloadKernelPackage}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	return []pulumi.Resource{extractPackageDone}, nil
}

func copyKernelHeaders(runner *command.Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	permissionFixArgs := command.Args{
		Create: pulumi.String("chown -R libvirt-qemu:kvm /tmp/kernel-packages/kernel-v*.pkg"),
		Sudo:   true,
	}
	permissionFixDone, err := runner.Command("permission-fix-kernel-headers", &permissionFixArgs, pulumi.DependsOn(depends))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	copyKernelHeadersArgs := command.Args{
		Create: pulumi.Sprintf(
			"-u libvirt-qemu /bin/bash -c \"cd /tmp; find /tmp/kernel-packages -name 'linux-image-*' -type f | xargs -i cp {} %s && find /tmp/kernel-packages -name 'linux-headers-*' -type f | xargs -i cp {} %s\"", kernelHeadersDir, kernelHeadersDir,
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

func prepareLibvirtSSHKeys(runner *command.Runner, localRunner *command.LocalRunner, resourceNamer namer.Namer, arch, tempDir string, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	privateKeyPath := filepath.Join(tempDir, fmt.Sprintf(libvirtSSHPrivateKey, arch))
	publicKeyPath := filepath.Join(tempDir, fmt.Sprintf(libvirtSSHPublicKey, arch))
	sshGenArgs := command.Args{
		Create: pulumi.Sprintf("rm -f %s && rm -f %s && ssh-keygen -t rsa -b 4096 -f %s -q -N \"\" && cat %s", privateKeyPath, publicKeyPath, privateKeyPath, publicKeyPath),
		Delete: pulumi.Sprintf("rm %s && rm %s", privateKeyPath, publicKeyPath),
	}
	sshgenDone, err := localRunner.Command(resourceNamer.ResourceName("gen-libvirt-sshkey"), &sshGenArgs)
	if err != nil {
		return []pulumi.Resource{}, err
	}

	sshWriteArgs := command.Args{
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
func provisionInstance(instance *Instance, m *sconfig.DDMicroVMConfig) ([]pulumi.Resource, error) {
	runner := instance.remoteRunner
	localRunner := instance.localRunner
	resourceNamer := instance.instanceNamer

	packagesInstallDone, err := installPackages(runner)
	if err != nil {
		return []pulumi.Resource{}, err
	}

	prepareLibvirtEnvDone, err := prepareLibvirtEnvironment(runner, packagesInstallDone)
	if err != nil {
		return []pulumi.Resource{}, err
	}

	downloadKernelDone, err := downloadAndExtractKernelPackage(instance.remoteRunner, instance.Arch)
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

	dependencies := append(downloadKernelDone, buildKernelHeadersDirDone)
	copyKernelHeadersDone, err := copyKernelHeaders(runner, dependencies)
	if err != nil {
		return []pulumi.Resource{}, err
	}

	tempDir := m.GetStringWithDefault(m.MicroVMConfig, "tempDir", "/tmp")
	prepareSSHKeysDone, err := prepareLibvirtSSHKeys(runner, localRunner, resourceNamer, instance.Arch, tempDir, []pulumi.Resource{})
	if err != nil {
		return []pulumi.Resource{}, err
	}

	return append(prepareSSHKeysDone, copyKernelHeadersDone...), nil

}
