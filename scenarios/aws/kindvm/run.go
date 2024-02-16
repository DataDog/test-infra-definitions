package kindvm

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/datadog/agent"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/cpustress"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/dogstatsd"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/mutatedbyadmissioncontroller"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/nginx"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/prometheus"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/redis"
	dogstatsdstandalone "github.com/DataDog/test-infra-definitions/components/datadog/dogstatsd-standalone"
	fakeintakeComp "github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
	localKubernetes "github.com/DataDog/test-infra-definitions/components/kubernetes"
	resAws "github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/ec2"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/fakeintake"

	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Run(ctx *pulumi.Context) error {
	awsEnv, err := resAws.NewEnvironment(ctx)
	if err != nil {
		return err
	}

	vm, err := ec2.NewVM(awsEnv, "kind")
	if err != nil {
		return err
	}
	if err := vm.Export(ctx, nil); err != nil {
		return err
	}

	kindClusterName := ctx.Stack()

	kindCluster, err := localKubernetes.NewKindCluster(*awsEnv.CommonEnvironment, vm, awsEnv.CommonNamer.ResourceName("kind"), kindClusterName, awsEnv.KubernetesVersion())
	if err != nil {
		return err
	}
	if err := kindCluster.Export(ctx, nil); err != nil {
		return err
	}

	// Building Kubernetes provider
	kindKubeProvider, err := kubernetes.NewProvider(ctx, awsEnv.Namer.ResourceName("k8s-provider"), &kubernetes.ProviderArgs{
		EnableServerSideApply: pulumi.BoolPtr(true),
		Kubeconfig:            kindCluster.KubeConfig,
	})
	if err != nil {
		return err
	}

	var dependsOnCrd pulumi.ResourceOption

	var fakeIntake *fakeintakeComp.Fakeintake
	if awsEnv.GetCommonEnvironment().AgentUseFakeintake() {
		fakeIntakeOptions := []fakeintake.Option{
			fakeintake.WithCPU(1024),
			fakeintake.WithMemory(6144),
		}
		if awsEnv.GetCommonEnvironment().InfraShouldDeployFakeintakeWithLB() {
			fakeIntakeOptions = append(fakeIntakeOptions, fakeintake.WithLoadBalancer())
		}

		if fakeIntake, err = fakeintake.NewECSFargateInstance(awsEnv, kindCluster.Name(), fakeIntakeOptions...); err != nil {
			return err
		}
		if err := fakeIntake.Export(awsEnv.Ctx, nil); err != nil {
			return err
		}
	}

	// Deploy the agent
	if awsEnv.AgentDeploy() {
		customValues := fmt.Sprintf(`
datadog:
  kubelet:
    tlsVerify: false
  clusterName: "%s"
agents:
  useHostNetwork: true
`, kindClusterName)

		agentFullImagePath := ""
		clusterAgentFullImagePath := ""

		if awsEnv.PipelineID() != "" && awsEnv.CommitSHA() != "" {
			agentFullImagePath = fmt.Sprintf("%s/agent:%s-%s", awsEnv.DefaultInternalRegistry(), awsEnv.PipelineID(), awsEnv.CommitSHA())
			clusterAgentFullImagePath = fmt.Sprintf("%s/cluster-agent:%s-%s", awsEnv.DefaultInternalRegistry(), awsEnv.PipelineID(), awsEnv.CommitSHA())
		}

		helmComponent, err := agent.NewHelmInstallation(*awsEnv.CommonEnvironment, agent.HelmInstallationArgs{
			ClusterAgentFullImagePath: clusterAgentFullImagePath,
			AgentFullImagePath:        agentFullImagePath,
			KubeProvider:              kindKubeProvider,
			Namespace:                 "datadog",
			ValuesYAML: pulumi.AssetOrArchiveArray{
				pulumi.NewStringAsset(customValues),
			},
			Fakeintake: fakeIntake,
		}, nil)
		if err != nil {
			return err
		}

		ctx.Export("agent-linux-helm-install-name", helmComponent.LinuxHelmReleaseName)
		ctx.Export("agent-linux-helm-install-status", helmComponent.LinuxHelmReleaseStatus)

		dependsOnCrd = utils.PulumiDependsOn(helmComponent)
	}

	// Deploy standalone dogstatsd
	if awsEnv.DogstatsdDeploy() {
		dogstatsdFullImagePath := ""
		if awsEnv.CommonEnvironment.PipelineID() != "" && awsEnv.CommonEnvironment.CommitSHA() != "" {
			dogstatsdFullImagePath = fmt.Sprintf("%s:%s-%s", awsEnv.DefaultInternalRegistry(), awsEnv.CommonEnvironment.PipelineID(), awsEnv.CommonEnvironment.CommitSHA())
		}

		if _, err := dogstatsdstandalone.K8sAppDefinition(*awsEnv.CommonEnvironment, kindKubeProvider, "dogstatsd-standalone", fakeIntake, false, kindClusterName, dogstatsdFullImagePath); err != nil {
			return err
		}
	}

	// Deploy testing workload
	if awsEnv.TestingWorkloadDeploy() {
		if _, err := nginx.K8sAppDefinition(*awsEnv.CommonEnvironment, kindKubeProvider, "workload-nginx", dependsOnCrd); err != nil {
			return err
		}

		if _, err := redis.K8sAppDefinition(*awsEnv.CommonEnvironment, kindKubeProvider, "workload-redis", dependsOnCrd); err != nil {
			return err
		}

		if _, err := cpustress.K8sAppDefinition(*awsEnv.CommonEnvironment, kindKubeProvider, "workload-cpustress"); err != nil {
			return err
		}

		// dogstatsd clients that report to the Agent
		if _, err := dogstatsd.K8sAppDefinition(*awsEnv.CommonEnvironment, kindKubeProvider, "workload-dogstatsd", 8125, "/var/run/datadog/dsd.socket"); err != nil {
			return err
		}

		// dogstatsd clients that report to the dogstatsd standalone deployment
		if _, err := dogstatsd.K8sAppDefinition(*awsEnv.CommonEnvironment, kindKubeProvider, "workload-dogstatsd-standalone", dogstatsdstandalone.HostPort, dogstatsdstandalone.Socket); err != nil {
			return err
		}

		if _, err := prometheus.K8sAppDefinition(*awsEnv.CommonEnvironment, kindKubeProvider, "workload-prometheus"); err != nil {
			return err
		}

		if _, err := mutatedbyadmissioncontroller.K8sAppDefinition(*awsEnv.CommonEnvironment, kindKubeProvider, "workload-mutated"); err != nil {
			return err
		}
	}

	return nil
}
