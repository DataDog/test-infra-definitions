package kubernetes

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components"
	"github.com/DataDog/test-infra-definitions/components/command"

	// "github.com/DataDog/test-infra-definitions/components/remote"
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
			Create: pulumi.String("crc setup"),
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

// func NewCrcClusterConfig(env config.Env, vm *remote.Host, name string, opts ...pulumi.ResourceOption) (*Cluster, error) {
//     return components.NewComponent(env, name, func(clusterComp *Cluster) error {
//         crcClusterName := env.CommonNamer().DisplayName(49)
//         opts = utils.MergeOptions[pulumi.ResourceOption](opts, pulumi.Parent(clusterComp))
//         runner := vm.OS.Runner()
//         commonEnvironment := env
//     }
