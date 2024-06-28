package kindvm

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/datadog/agent"
	"github.com/DataDog/test-infra-definitions/components/datadog/agent/helm"
	"github.com/DataDog/test-infra-definitions/components/datadog/agentwithoperatorparams"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/cpustress"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/dogstatsd"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/mutatedbyadmissioncontroller"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/nginx"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/prometheus"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/redis"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/tracegen"
	dogstatsdstandalone "github.com/DataDog/test-infra-definitions/components/datadog/dogstatsd-standalone"
	fakeintakeComp "github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
	"github.com/DataDog/test-infra-definitions/components/datadog/kubernetesagentparams"
	"github.com/DataDog/test-infra-definitions/components/datadog/operatorparams"
	localKubernetes "github.com/DataDog/test-infra-definitions/components/kubernetes"
	"github.com/DataDog/test-infra-definitions/components/os"
	resAws "github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/ec2"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/fakeintake"

	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Run(ctx *pulumi.Context) error {
	awsEnv, err := resAws.NewEnvironment(ctx)
	if err != nil {
		return err
	}

	osDesc := os.DescriptorFromString(awsEnv.InfraOSDescriptor(), os.AmazonLinuxECSDefault)
	vm, err := ec2.NewVM(awsEnv, "kind", ec2.WithOS(osDesc))
	if err != nil {
		return err
	}
	if err := vm.Export(ctx, nil); err != nil {
		return err
	}

	kindClusterName := ctx.Stack()

	installEcrCredsHelperCmd, err := ec2.InstallECRCredentialsHelper(awsEnv, vm)
	if err != nil {
		return err
	}

	kindCluster, err := localKubernetes.NewKindCluster(&awsEnv, vm, awsEnv.CommonNamer().ResourceName("kind"), kindClusterName, awsEnv.KubernetesVersion(), utils.PulumiDependsOn(installEcrCredsHelperCmd))
	if err != nil {
		return err
	}
	if err := kindCluster.Export(ctx, nil); err != nil {
		return err
	}

	// Building Kubernetes provider
	kindKubeProvider, err := kubernetes.NewProvider(ctx, awsEnv.Namer.ResourceName("k8s-provider"), &kubernetes.ProviderArgs{
		Kubeconfig:            kindCluster.KubeConfig,
		EnableServerSideApply: pulumi.BoolPtr(true),
		DeleteUnreachable:     pulumi.BoolPtr(true),
	})
	if err != nil {
		return err
	}

	var dependsOnCrd pulumi.ResourceOption

	var fakeIntake *fakeintakeComp.Fakeintake
	if awsEnv.AgentUseFakeintake() {
		fakeIntakeOptions := []fakeintake.Option{
			fakeintake.WithMemory(2048),
		}
		if awsEnv.InfraShouldDeployFakeintakeWithLB() {
			fakeIntakeOptions = append(fakeIntakeOptions, fakeintake.WithLoadBalancer())
		}

		if fakeIntake, err = fakeintake.NewECSFargateInstance(awsEnv, kindCluster.Name(), fakeIntakeOptions...); err != nil {
			return err
		}

		if err := fakeIntake.Export(awsEnv.Ctx(), nil); err != nil {
			return err
		}
	}

	// Deploy the agent
	if awsEnv.AgentDeploy() && !awsEnv.AgentDeployWithOperator() {
		customValues := fmt.Sprintf(`
datadog:
  kubelet:
    tlsVerify: false
  clusterName: "%s"
agents:
  useHostNetwork: true
`, kindClusterName)

		k8sAgentOptions := make([]kubernetesagentparams.Option, 0)
		k8sAgentOptions = append(
			k8sAgentOptions,
			kubernetesagentparams.WithNamespace("datadog"),
			kubernetesagentparams.WithHelmValues(customValues),
		)
		if fakeIntake != nil {
			k8sAgentOptions = append(
				k8sAgentOptions,
				kubernetesagentparams.WithFakeintake(fakeIntake),
			)
		}

		k8sAgentComponent, err := helm.NewKubernetesAgent(&awsEnv, awsEnv.Namer.ResourceName("datadog-agent"), kindKubeProvider, k8sAgentOptions...)

		if err != nil {
			return err
		}

		if err := k8sAgentComponent.Export(awsEnv.Ctx(), nil); err != nil {
			return err
		}

		dependsOnCrd = utils.PulumiDependsOn(k8sAgentComponent)
	}

	// Deploy the operator
	if awsEnv.AgentDeploy() && awsEnv.AgentDeployWithOperator() {
		operatorOpts := make([]operatorparams.Option, 0)
		operatorOpts = append(
			operatorOpts,
			operatorparams.WithNamespace("datadog"),
			operatorparams.WithFakeIntake(fakeIntake),
		)
		ddaOptions := make([]agentwithoperatorparams.Option, 0)
		ddaOptions = append(
			ddaOptions,
			agentwithoperatorparams.WithNamespace("datadog"),
			agentwithoperatorparams.WithTLSKubeletVerify(false),
		)

		operatorAgentComponent, err := agent.NewDDAWithOperator(&awsEnv, awsEnv.CommonNamer().ResourceName("dd-operator-agent"), kindKubeProvider, operatorOpts, ddaOptions...)
		if err != nil {
			return err
		}

		dependsOnCrd = utils.PulumiDependsOn(operatorAgentComponent)

		if err := operatorAgentComponent.Export(awsEnv.Ctx(), nil); err != nil {
			return err
		}
	}

	// Deploy standalone dogstatsd
	if awsEnv.DogstatsdDeploy() {
		if _, err := dogstatsdstandalone.K8sAppDefinition(&awsEnv, kindKubeProvider, "dogstatsd-standalone", fakeIntake, false, kindClusterName); err != nil {
			return err
		}
	}

	// Deploy testing workload
	if awsEnv.TestingWorkloadDeploy() {
		if _, err := nginx.K8sAppDefinition(&awsEnv, kindKubeProvider, "workload-nginx", "", true, dependsOnCrd); err != nil {
			return err
		}

		if _, err := redis.K8sAppDefinition(&awsEnv, kindKubeProvider, "workload-redis", true, dependsOnCrd); err != nil {
			return err
		}

		if _, err := cpustress.K8sAppDefinition(&awsEnv, kindKubeProvider, "workload-cpustress"); err != nil {
			return err
		}

		// dogstatsd clients that report to the Agent
		if _, err := dogstatsd.K8sAppDefinition(&awsEnv, kindKubeProvider, "workload-dogstatsd", 8125, "/var/run/datadog/dsd.socket"); err != nil {
			return err
		}

		// dogstatsd clients that report to the dogstatsd standalone deployment
		if _, err := dogstatsd.K8sAppDefinition(&awsEnv, kindKubeProvider, "workload-dogstatsd-standalone", dogstatsdstandalone.HostPort, dogstatsdstandalone.Socket); err != nil {
			return err
		}

		// for tracegen we can't find the cgroup version as it depends on the underlying version of the kernel
		if _, err := tracegen.K8sAppDefinition(&awsEnv, kindKubeProvider, "workload-tracegen"); err != nil {
			return err
		}

		if _, err := prometheus.K8sAppDefinition(&awsEnv, kindKubeProvider, "workload-prometheus"); err != nil {
			return err
		}

		if _, err := mutatedbyadmissioncontroller.K8sAppDefinition(&awsEnv, kindKubeProvider, "workload-mutated", "workload-mutated-lib-injection"); err != nil {
			return err
		}
	}

	return nil
}
