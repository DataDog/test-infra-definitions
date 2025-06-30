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

func NewLocalOpenShiftCluster(env config.Env, name string, opts ...pulumi.ResourceOption) (*Cluster, error) {
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
			Create: pulumi.String("crc setup"),
		}, opts...)
		if err != nil {
			return err
		}

		// Start CRC cluster with proper timeout
		startCluster, err := runner.Command(commonEnvironment.CommonNamer().ResourceName("crc-start"), &command.Args{
			Create: pulumi.Sprintf("timeout 1800 crc start -p %s", pullSecretPath), // 30 minute timeout
			Delete: pulumi.String("crc stop || true"),
			Triggers: pulumi.Array{
				pulumi.String(pullSecretPath),
			},
		}, utils.MergeOptions(opts, utils.PulumiDependsOn(crcSetup))...)
		if err != nil {
			return err
		}

		kubeConfigCmd, err := runner.Command(commonEnvironment.CommonNamer().ResourceName("get-kubeconfig"), &command.Args{
			Create: pulumi.String("cat ~/.crc/machines/crc/kubeconfig"),
		}, utils.MergeOptions(opts, utils.PulumiDependsOn(startCluster))...)
		if err != nil {
			return err
		}

		clusterComp.KubeConfig = kubeConfigCmd.StdoutOutput()
		clusterComp.ClusterName = pulumi.String("openshift").ToStringOutput()
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

		installLibvirt, err := runner.Command(commonEnvironment.CommonNamer().ResourceName("install-libvirt"), &command.Args{
			Create: pulumi.String(`
		sudo dnf install -y libvirt NetworkManager`),
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

		setNetworking, err := runner.Command("set-network-mode", &command.Args{
			Create: pulumi.String(`crc config set network-mode system`),
		}, utils.MergeOptions(opts, utils.PulumiDependsOn(restartServices))...)
		if err != nil {
			return err
		}

		setupCRC, err := runner.Command(commonEnvironment.CommonNamer().ResourceName("crc-setup"), &command.Args{
			Create: pulumi.String("crc cleanup && crc setup"),
		}, utils.MergeOptions(opts, utils.PulumiDependsOn(pullSecretFile, setNetworking))...)
		if err != nil {
			return err
		}
		//debugging purposes with pulumi verbose logs
		ensureCRCDaemon, err := runner.Command(commonEnvironment.CommonNamer().ResourceName("ensure-crc-daemon"), &command.Args{
			Create: pulumi.String(`
				systemctl --user enable crc-daemon.service && \
				systemctl --user start crc-daemon.service && \
				systemctl --user status crc-daemon.service
			`),
		}, utils.MergeOptions(opts, utils.PulumiDependsOn(setupCRC))...)
		if err != nil {
			return err
		}
		//debugging purposes with pulumi verbose logs
		startCRC, err := runner.Command(commonEnvironment.CommonNamer().ResourceName("crc-start"), &command.Args{
			Create: pulumi.String(`
				for i in {1..3}; do
					crc start -p /tmp/pull-secret.txt && break
					echo "crc start failed, retrying in 30s..."
					sleep 30
				done`),
			Delete: pulumi.String("crc stop && crc delete && crc cleanup && rm -rf ~/.crc"),
			Triggers: pulumi.Array{
				pulumi.String(pullSecretPath),
			},
		}, utils.MergeOptions(opts, utils.PulumiDependsOn(ensureCRCDaemon))...)
		if err != nil {
			return err
		}
		//debugging purposes with pulumi verbose logs
		waitForAPI, err := runner.Command("wait-for-api", &command.Args{
			Create: pulumi.String(`
				for i in {1..60}; do
					echo "Checking CRC status..."
					crc status
					if ! crc status | grep -q "Disk Usage: 0B"; then
						if curl -k https://api.crc.testing:6443/healthz; then
							echo "API is up!"
							exit 0
						else
							echo "API not yet up, sleeping..."
						fi
					else
						echo "CRC VM is up but disk not ready — waiting..."
					fi
					sleep 10
				done
				echo "API never became ready"
				exit 1
			`),
		}, utils.MergeOptions(opts, utils.PulumiDependsOn(startCRC))...)
		if err != nil {
			return err
		}

		kubeConfig, err := runner.Command(commonEnvironment.CommonNamer().ResourceName("get-kubeconfig"), &command.Args{
			Create: pulumi.String("cat ~/.crc/machines/crc/kubeconfig"),
		}, utils.MergeOptions(opts, utils.PulumiDependsOn(waitForAPI))...)
		if err != nil {
			return err
		}

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
			Create: pulumi.Sprintf(`curl --retry 10 -fsSL https://developers.redhat.com/content-gateway/file/pub/openshift-v4/clients/crc/2.52.0/crc-linux-%s.tar.xz -o crc.tar.xz && \
	tar -xf crc.tar.xz && \
	sudo mv crc-linux-*-%s/crc /usr/local/bin/crc`, openShiftArch, openShiftArch),
		}, opts...)
}
