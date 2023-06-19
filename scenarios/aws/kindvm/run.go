package kindvm

import (
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/datadog/agent"
	localKubernetes "github.com/DataDog/test-infra-definitions/components/kubernetes"
	ec2vm "github.com/DataDog/test-infra-definitions/scenarios/aws/vm/ec2VM"

	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Run(ctx *pulumi.Context) error {
	vm, err := ec2vm.NewUnixEc2VM(ctx)
	if err != nil {
		return err
	}
	awsEnv := vm.Infra.GetAwsEnvironment()

	kubeConfigCommand, kubeConfig, err := localKubernetes.NewKindCluster(vm.UnixVM, awsEnv.CommonNamer.ResourceName("kind"), "amd64")
	if err != nil {
		return err
	}

	// Building Kubernetes provider
	kindKubeProvider, err := kubernetes.NewProvider(ctx, awsEnv.Namer.ResourceName("k8s-provider"), &kubernetes.ProviderArgs{
		EnableServerSideApply: pulumi.BoolPtr(true),
		Kubeconfig:            kubeConfig,
	}, utils.PulumiDependsOn(kubeConfigCommand))
	if err != nil {
		return err
	}

	if awsEnv.AgentDeploy() {
		customValues := `
datadog:
  kubelet:
    tlsVerify: false
`

		_, err := agent.NewHelmInstallation(*awsEnv.CommonEnvironment, agent.HelmInstallationArgs{
			KubeProvider: kindKubeProvider,
			Namespace:    "datadog",
			ValuesYAML: pulumi.AssetOrArchiveArray{
				pulumi.NewStringAsset(customValues),
			},
		}, nil)
		if err != nil {
			return err
		}
	}

	ctx.Export("kubeconfig", kubeConfig)
	return err
}
