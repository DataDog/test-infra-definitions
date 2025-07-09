package openshiftvm

import (
	helm "github.com/DataDog/test-infra-definitions/components/datadog/agent/helm"
	kubernetesagentparams "github.com/DataDog/test-infra-definitions/components/datadog/kubernetesagentparams"
	localKubernetes "github.com/DataDog/test-infra-definitions/components/kubernetes"
	"github.com/DataDog/test-infra-definitions/components/os"
	resGcp "github.com/DataDog/test-infra-definitions/resources/gcp"
	"github.com/DataDog/test-infra-definitions/scenarios/gcp/compute"
	kubernetes "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Run(ctx *pulumi.Context) error {
	gcpEnv, err := resGcp.NewEnvironment(ctx)
	if err != nil {
		return err
	}

	osDesc := os.DescriptorFromString("redhat:9", os.RedHat9)
	vm, err := compute.NewVM(gcpEnv, "openshift",
		compute.WithOS(osDesc),
		compute.WithInstanceType("n2-standard-8"))
	if err != nil {
		return err
	}
	if err := vm.Export(ctx, nil); err != nil {
		return err
	}

	openshiftCluster, err := localKubernetes.NewOpenShiftCluster(&gcpEnv, vm, "openshift")
	if err != nil {
		return err
	}
	if err := openshiftCluster.Export(ctx, nil); err != nil {
		return err
	}

	// Building Kubernetes provider for OpenShift
	openshiftKubeProvider, err := kubernetes.NewProvider(ctx, gcpEnv.Namer.ResourceName("openshift-k8s-provider"), &kubernetes.ProviderArgs{
		Kubeconfig:            openshiftCluster.KubeConfig,
		EnableServerSideApply: pulumi.BoolPtr(true),
		DeleteUnreachable:     pulumi.BoolPtr(true),
	})
	if err != nil {
		return err
	}

	// Deploy the Datadog agent via Helm
	if gcpEnv.AgentDeploy() && !gcpEnv.AgentDeployWithOperator() {
		customValues := `
    datadog:
      kubelet:
        tlsVerify: false
    agents:
      useHostNetwork: true
    `

		k8sAgentOptions := make([]kubernetesagentparams.Option, 0)
		k8sAgentOptions = append(
			k8sAgentOptions,
			kubernetesagentparams.WithNamespace("datadog"),
			kubernetesagentparams.WithHelmValues(customValues),
		)

		k8sAgentComponent, err := helm.NewKubernetesAgent(&gcpEnv, gcpEnv.Namer.ResourceName("datadog-agent"), openshiftKubeProvider, k8sAgentOptions...)
		if err != nil {
			return err
		}

		if err := k8sAgentComponent.Export(gcpEnv.Ctx(), nil); err != nil {
			return err
		}
	}

	return nil
}
