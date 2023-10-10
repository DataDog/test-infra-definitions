package kindvm

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/datadog/agent"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/dogstatsd"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/nginx"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/prometheus"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/redis"
	ddfakeintake "github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
	localKubernetes "github.com/DataDog/test-infra-definitions/components/kubernetes"
	"github.com/DataDog/test-infra-definitions/scenarios/aws"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/vm/ec2vm"

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

	// Export cluster’s properties
	ctx.Export("kubeconfig", kubeConfig)

	// Building Kubernetes provider
	kindKubeProvider, err := kubernetes.NewProvider(ctx, awsEnv.Namer.ResourceName("k8s-provider"), &kubernetes.ProviderArgs{
		EnableServerSideApply: pulumi.BoolPtr(true),
		Kubeconfig:            kubeConfig,
	}, utils.PulumiDependsOn(kubeConfigCommand))
	if err != nil {
		return err
	}

	var dependsOnCrd pulumi.ResourceOption

	// Deploy the agent
	if awsEnv.AgentDeploy() {
		var fakeintake *ddfakeintake.ConnectionExporter
		if awsEnv.GetCommonEnvironment().AgentUseFakeintake() {
			if fakeintake, err = aws.NewEcsFakeintake(awsEnv); err != nil {
				return err
			}
		}
		customValues := fmt.Sprintf(`
datadog:
  kubelet:
    tlsVerify: false
  clusterName: "%s"
`, ctx.Stack())

		helmComponent, err := agent.NewHelmInstallation(*awsEnv.CommonEnvironment, agent.HelmInstallationArgs{
			KubeProvider: kindKubeProvider,
			Namespace:    "datadog",
			ValuesYAML: pulumi.AssetOrArchiveArray{
				pulumi.NewStringAsset(customValues),
			},
			Fakeintake: fakeintake,
		}, nil)
		if err != nil {
			return err
		}

		ctx.Export("kube-cluster-name", pulumi.String(ctx.Stack()))
		ctx.Export("agent-linux-helm-install-name", helmComponent.LinuxHelmReleaseName)
		ctx.Export("agent-linux-helm-install-status", helmComponent.LinuxHelmReleaseStatus)

		dependsOnCrd = utils.PulumiDependsOn(helmComponent)
	}

	// Deploy testing workload
	if awsEnv.TestingWorkloadDeploy() {
		if _, err := nginx.K8sAppDefinition(*awsEnv.CommonEnvironment, kindKubeProvider, "workload-nginx", dependsOnCrd); err != nil {
			return err
		}

		if _, err := redis.K8sAppDefinition(*awsEnv.CommonEnvironment, kindKubeProvider, "workload-redis", dependsOnCrd); err != nil {
			return err
		}

		if _, err := dogstatsd.K8sAppDefinition(*awsEnv.CommonEnvironment, kindKubeProvider, "workload-dogstatsd"); err != nil {
			return err
		}

		if _, err := prometheus.K8sAppDefinition(*awsEnv.CommonEnvironment, kindKubeProvider, "workload-prometheus"); err != nil {
			return err
		}
	}

	return nil
}
