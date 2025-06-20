package kubernetes

import (
	"fmt"
	"os"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components"
	"github.com/DataDog/test-infra-definitions/components/command"
	oscomp "github.com/DataDog/test-infra-definitions/components/os"
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

		pullSecretPath := os.Getenv("PULL_SECRET_PATH")
		if pullSecretPath == "" {
			return fmt.Errorf("PULL_SECRET_PATH environment variable is not set")
		}

		crcSetup, err := runner.Command(commonEnvironment.CommonNamer().ResourceName("crc-setup"), &command.Args{
			Create: pulumi.String("crc setup --log-level debug"),
		}, opts...)
		if err != nil {
			return err
		}

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

func NewOpenShiftCluster(env config.Env, vm *remote.Host, name string, opts ...pulumi.ResourceOption) (*Cluster, error) {
	pullSecretPath := os.Getenv("PULL_SECRET_PATH")
	if pullSecretPath == "" {
		return nil, fmt.Errorf("PULL_SECRET_PATH environment variable is not set")
	}

	return components.NewComponent(env, name, func(clusterComp *Cluster) error {
		openShiftClusterName := env.CommonNamer().DisplayName(49)
		opts = utils.MergeOptions[pulumi.ResourceOption](opts, pulumi.Parent(clusterComp))
		runner := vm.OS.Runner()
		commonEnvironment := env

		openShiftInstallBinary, err := InstallOpenShiftBinary(env, vm, opts...)
		if err != nil {
			return err
		}

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

		// https://documentation.ubuntu.com/server/how-to/virtualisation/libvirt/
		installLibvirt, err := runner.Command(commonEnvironment.CommonNamer().ResourceName("install-libvirt"), &command.Args{
			Create: pulumi.String(`
		sudo apt update && \
		sudo apt install -y qemu-kvm libvirt-daemon-system && \
		sudo adduser gce libvirt && \
		newgrp libvirt`),
		}, utils.MergeOptions(opts, utils.PulumiDependsOn(openShiftInstallBinary))...)
		if err != nil {
			return err
		}
		// https://medium.com/@python-javascript-php-html-css/troubleshooting-ssh-handshake-failed-error-on-openshift-codeready-containers-6bdd1cf08bbb
		restartServices, err := runner.Command(commonEnvironment.CommonNamer().ResourceName("restart-services"), &command.Args{
			Create: pulumi.String(`sudo systemctl restart libvirtd  && sudo systemctl restart sshd`),
		}, utils.MergeOptions(opts, utils.PulumiDependsOn(installLibvirt))...)
		if err != nil {
			return err
		}

		setupCRC, err := runner.Command(commonEnvironment.CommonNamer().ResourceName("crc-setup"), &command.Args{
			Create: pulumi.String("crc setup --log-level debug"),
		}, utils.MergeOptions(opts, utils.PulumiDependsOn(pullSecretFile, restartServices))...)
		if err != nil {
			return err
		}

		startCRC, err := runner.Command(commonEnvironment.CommonNamer().ResourceName("crc-start"), &command.Args{
			Create: pulumi.Sprintf("crc start -p /tmp/pull-secret.txt --log-level debug"),
			Delete: pulumi.String("crc stop"),
		}, utils.MergeOptions(opts, utils.PulumiDependsOn(setupCRC))...)
		if err != nil {
			return err
		}

		kubeConfig, err := runner.Command(commonEnvironment.CommonNamer().ResourceName("get-kubeconfig"), &command.Args{
			Create: pulumi.String("cat ~/.crc/machines/crc/kubeconfig"),
		}, utils.MergeOptions(opts, utils.PulumiDependsOn(startCRC))...)
		if err != nil {
			return err
		}

		installKubectl, err := runner.Command(commonEnvironment.CommonNamer().ResourceName("install-kubectl"), &command.Args{
			Create: pulumi.String(`curl -LO "https://dl.k8s.io/release/v1.28.0/bin/linux/amd64/kubectl" && chmod +x kubectl && sudo mv kubectl /usr/local/bin/`),
		}, utils.MergeOptions(opts, utils.PulumiDependsOn(kubeConfig))...)
		if err != nil {
			return err
		}

		switchContext, err := runner.Command(commonEnvironment.CommonNamer().ResourceName("kubectl-use-context"), &command.Args{
			Create: pulumi.String(`kubectl config use-context crc-admin`),
		}, utils.MergeOptions(opts, utils.PulumiDependsOn(installKubectl))...)
		if err != nil {
			return fmt.Errorf("failed to switch kubectl context to crc-admin: %w", err)
		}

		verifyCluster, err := runner.Command(commonEnvironment.CommonNamer().ResourceName("verify-openshift"), &command.Args{
			Create: pulumi.String(`kubectl get nodes`),
		}, utils.MergeOptions(opts, utils.PulumiDependsOn(switchContext))...)
		if err != nil {
			return fmt.Errorf("OpenShift cluster is not responding to kubectl get nodes: %w", err)
		}
		clusterComp.CRCVerifyLog = verifyCluster.StdoutOutput()
		clusterComp.CRCStartLog = startCRC.StderrOutput().ToStringOutput()
		clusterComp.KubeConfig = kubeConfig.StdoutOutput()
		clusterComp.ClusterName = openShiftClusterName.ToStringOutput()
		return nil
	}, opts...)
}

func InstallOpenShiftBinary(env config.Env, vm *remote.Host, opts ...pulumi.ResourceOption) (pulumi.Resource, error) {
	openShiftArch := vm.OS.Descriptor().Architecture
	if openShiftArch == oscomp.AMD64Arch {
		openShiftArch = "amd64"
	}
	return vm.OS.Runner().Command(
		env.CommonNamer().ResourceName("crc-install"),
		&command.Args{
			Create: pulumi.Sprintf(`curl --retry 10 -fsSL https://mirror.openshift.com/pub/openshift-v4/clients/crc/latest/crc-linux-%s.tar.xz -o crc.tar.xz && \
	tar -xf crc.tar.xz && \
	sudo mv crc-linux-*-%s/crc /usr/local/bin/crc`, openShiftArch, openShiftArch),
		}, opts...)
}
