package eks

import (
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/datadog/agent/helm"
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
	resourcesAws "github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/fakeintake"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Run(ctx *pulumi.Context) error {
	awsEnv, err := resourcesAws.NewEnvironment(ctx)
	if err != nil {
		return err
	}

	clusterOptions := buildClusterOptionsFromConfigMap(awsEnv)

	cluster, err := NewCluster(awsEnv, "eks", clusterOptions...)
	if err != nil {
		return err
	}

	err = cluster.Export(ctx, nil)
	if err != nil {
		return err
	}

	if awsEnv.InitOnly() {
		return nil
	}

	// Create fakeintake if needed
	var fakeIntake *fakeintakeComp.Fakeintake
	if awsEnv.AgentUseFakeintake() {
		fakeIntakeOptions := []fakeintake.Option{
			fakeintake.WithMemory(2048),
		}
		if awsEnv.InfraShouldDeployFakeintakeWithLB() {
			fakeIntakeOptions = append(fakeIntakeOptions, fakeintake.WithLoadBalancer())
		}

		if fakeIntake, err = fakeintake.NewECSFargateInstance(awsEnv, "ecs", fakeIntakeOptions...); err != nil {
			return err
		}
		if err := fakeIntake.Export(awsEnv.Ctx(), nil); err != nil {
			return err
		}
	}
	workloadWithCRDDeps := make([]pulumi.Resource, 0)
	if awsEnv.AgentDeploy() {
		k8sAgentOptions := make([]kubernetesagentparams.Option, 0)
		k8sAgentOptions = append(
			k8sAgentOptions,
			kubernetesagentparams.WithNamespace("datadog"),
			kubernetesagentparams.WithPulumiResourceOptions(utils.PulumiDependsOn(cluster)),
		)

		if awsEnv.AgentUseFakeintake() {
			k8sAgentOptions = append(k8sAgentOptions, kubernetesagentparams.WithFakeintake(fakeIntake))
		}

		if awsEnv.EKSWindowsNodeGroup() {
			k8sAgentOptions = append(k8sAgentOptions, kubernetesagentparams.WithDeployWindows())
		}

		k8sAgentComponent, err := helm.NewKubernetesAgent(&awsEnv, awsEnv.Namer.ResourceName("datadog-agent"), cluster.KubeProvider, k8sAgentOptions...)

		if err != nil {
			return err
		}

		if err := k8sAgentComponent.Export(awsEnv.Ctx(), nil); err != nil {
			return err
		}

		workloadWithCRDDeps = append(workloadWithCRDDeps, k8sAgentComponent)

	}
	// Deploy standalone dogstatsd
	if awsEnv.DogstatsdDeploy() {
		if _, err := dogstatsdstandalone.K8sAppDefinition(&awsEnv, cluster.KubeProvider, "dogstatsd-standalone", fakeIntake, true, "", utils.PulumiDependsOn(cluster)); err != nil {
			return err
		}
	}

	// Deploy testing workload
	if awsEnv.TestingWorkloadDeploy() {
		if _, err := nginx.K8sAppDefinition(&awsEnv, cluster.KubeProvider, "workload-nginx", "", true, utils.PulumiDependsOn(workloadWithCRDDeps...)); err != nil {
			return err
		}

		if _, err := redis.K8sAppDefinition(&awsEnv, cluster.KubeProvider, "workload-redis", true, utils.PulumiDependsOn(workloadWithCRDDeps...)); err != nil {
			return err
		}

		if _, err := cpustress.K8sAppDefinition(&awsEnv, cluster.KubeProvider, "workload-cpustress", utils.PulumiDependsOn(cluster)); err != nil {
			return err
		}

		// dogstatsd clients that report to the Agent
		if _, err := dogstatsd.K8sAppDefinition(&awsEnv, cluster.KubeProvider, "workload-dogstatsd", 8125, "/var/run/datadog/dsd.socket", utils.PulumiDependsOn(cluster)); err != nil {
			return err
		}

		// dogstatsd clients that report to the dogstatsd standalone deployment
		if _, err := dogstatsd.K8sAppDefinition(&awsEnv, cluster.KubeProvider, "workload-dogstatsd-standalone", dogstatsdstandalone.HostPort, dogstatsdstandalone.Socket, utils.PulumiDependsOn(cluster)); err != nil {
			return err
		}

		if _, err := tracegen.K8sAppDefinition(&awsEnv, cluster.KubeProvider, "workload-tracegen", utils.PulumiDependsOn(cluster)); err != nil {
			return err
		}

		if _, err := prometheus.K8sAppDefinition(&awsEnv, cluster.KubeProvider, "workload-prometheus", utils.PulumiDependsOn(cluster)); err != nil {
			return err
		}

		if _, err := mutatedbyadmissioncontroller.K8sAppDefinition(&awsEnv, cluster.KubeProvider, "workload-mutated", "workload-mutated-lib-injection", utils.PulumiDependsOn(cluster)); err != nil {
			return err
		}
	}

	return nil
}
