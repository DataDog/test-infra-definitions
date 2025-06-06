package kubernetes

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/components/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func NewLocalCRCCluster(env config.Env, name string, opts ...pulumi.ResourceOption) (*Cluster, error) {
	return components.NewComponent(env, name, func(clusterComp *Cluster) error {
		opts = utils.MergeOptions[pulumi.ResourceOption](opts, pulumi.Parent(clusterComp))
		commonEnvironment := env
		runner := command.NewLocalRunner(env, command.LocalRunnerArgs{
			OSCommand: command.NewUnixOSCommand(),
		})

		// Local pull secret path
		pullSecretPath := "/Users/shaina.patel/Desktop/pull-secret.txt"

		//setup crc
		crcSetup, err := runner.Command(commonEnvironment.CommonNamer().ResourceName("crc-setup"), &command.Args{
			Create: pulumi.String("crc setup --log-level debug"),
		}, opts...)
		if err != nil {
			return err
		}

		//start a openshift local cluster
		startCluster, err := runner.Command(commonEnvironment.CommonNamer().ResourceName("crc-start"), &command.Args{
			Create: pulumi.Sprintf("crc start -p %s --log-level debug", pullSecretPath),
			Delete: pulumi.String("crc delete -f || true"),
			Triggers: pulumi.Array{
				pulumi.String(pullSecretPath),
			},
		}, utils.MergeOptions(opts, utils.PulumiDependsOn(crcSetup))...)
		if err != nil {
			return err
		}

		// Get kubeconfig
		kubeConfigCmd, err := runner.Command(commonEnvironment.CommonNamer().ResourceName("get-kubeconfig"), &command.Args{
			Create: pulumi.String("cat ~/.crc/cache/crc_vfkit_*/kubeconfig"),
		}, utils.MergeOptions(opts, utils.PulumiDependsOn(startCluster))...)
		if err != nil {
			return err
		}

		_, err = runner.Command(commonEnvironment.CommonNamer().ResourceName("keep-alive"), &command.Args{
			Create: pulumi.String("sleep 300"),
		}, utils.MergeOptions(opts, utils.PulumiDependsOn(kubeConfigCmd))...)
		if err != nil {
			return err
		}

		clusterComp.KubeConfig = kubeConfigCmd.StdoutOutput()
		clusterComp.ClusterName = pulumi.String("crc").ToStringOutput()
		return nil
	}, opts...)
}

func NewCrcCluster(env config.Env, vm *remote.Host, name string, opts ...pulumi.ResourceOption) (*Cluster, error) {
	pullSecretPath := "/Users/shaina.patel/Desktop/pull-secret.txt"

	return components.NewComponent(env, name, func(clusterComp *Cluster) error {
		opts = utils.MergeOptions[pulumi.ResourceOption](opts, pulumi.Parent(clusterComp))
		runner := vm.OS.Runner()
		commonEnvironment := env

		crcInstallBinary, err := InstallCRCBinary(env, vm, opts...)
		if err != nil {
			return err
		}

		// Read and copy pull secret file
		pullSecretContent, err := utils.ReadSecretFile(pullSecretPath)
		if err != nil {
			return err
		}
		pullSecretFile, err := vm.OS.FileManager().CopyInlineFile(
			pullSecretContent,
			"/tmp/pull-secret.txt",
		)
		if err != nil {
			return err
		}

		// make a bundle cache directory only
		prepareBundleDir, err := runner.Command(env.CommonNamer().ResourceName("prepare-bundle-dir"), &command.Args{
			Create: pulumi.String(`mkdir -p ~/.crc/cache`),
		}, utils.MergeOptions(opts, utils.PulumiDependsOn(crcInstallBinary))...)
		if err != nil {
			return err
		}

		// install libvirt and configure user groups I need this for nested virtualization to work
		// without all these commands I was getting errors like: Cannot get machine state: Unable to connect to kvm driver, did you add yourself to the libvirtd group
		// Come back to this and understand why these commands are needed / if I can remove some of them
		installLibvirt, err := runner.Command(commonEnvironment.CommonNamer().ResourceName("install-libvirt"), &command.Args{
			Create: pulumi.String(`sudo apt-get clean && \
		sudo apt-get update && \
		sudo apt-get install -y --fix-broken && \
		sudo apt-get install -y libvirt-daemon libvirt-daemon-system libvirt-clients && \
		sudo usermod -a -G libvirt gce && \
		sudo usermod -a -G libvirtd gce && \
		sudo systemctl enable --now libvirtd && \
		newgrp libvirtd || newgrp libvirt`),
		}, utils.MergeOptions(opts, utils.PulumiDependsOn(prepareBundleDir))...)
		if err != nil {
			return err
		}

		// run crc setup (auto-downloads bundle and prepares env)
		setupCRC, err := runner.Command(commonEnvironment.CommonNamer().ResourceName("crc-setup"), &command.Args{
			Create: pulumi.String("crc setup --log-level debug"),
		}, utils.MergeOptions(opts, utils.PulumiDependsOn(pullSecretFile, installLibvirt))...)
		if err != nil {
			return err
		}

		// start CRC with the pull secret
		startCRC, err := runner.Command(commonEnvironment.CommonNamer().ResourceName("crc-start"), &command.Args{
			Create: pulumi.Sprintf("crc start -p /tmp/pull-secret.txt --log-level debug"),
			Delete: pulumi.String("crc delete -f"),
		}, utils.MergeOptions(opts, utils.PulumiDependsOn(setupCRC))...)
		if err != nil {
			return err
		}

		// retrieve kubeconfig and verify status
		kubeConfig, err := runner.Command(commonEnvironment.CommonNamer().ResourceName("get-kubeconfig"), &command.Args{
			Create: pulumi.String("crc status --log-level debug && cat ~/.crc/machines/crc/kubeconfig"),
		}, utils.MergeOptions(opts, utils.PulumiDependsOn(startCRC))...)
		if err != nil {
			return err
		}

		clusterComp.KubeConfig = kubeConfig.StdoutOutput()
		clusterComp.ClusterName = pulumi.String("crc").ToStringOutput()
		return nil
	}, opts...)
}

func InstallCRCBinary(env config.Env, vm *remote.Host, opts ...pulumi.ResourceOption) (pulumi.Resource, error) {
	crcArch := vm.OS.Descriptor().Architecture
	if crcArch == os.AMD64Arch {
		crcArch = "amd64"
	}
	return vm.OS.Runner().Command(
		env.CommonNamer().ResourceName("crc-install"),
		&command.Args{
			Create: pulumi.Sprintf(`curl --retry 10 -fsSL https://mirror.openshift.com/pub/openshift-v4/clients/crc/latest/crc-linux-%s.tar.xz -o crc.tar.xz && \
	tar -xf crc.tar.xz && \
	sudo mv crc-linux-*-%s/crc /usr/local/bin/crc`, crcArch, crcArch),
		}, opts...)
}
