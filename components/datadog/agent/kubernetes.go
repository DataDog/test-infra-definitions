package agent

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/components"
	"github.com/DataDog/test-infra-definitions/components/datadog/kubernetesagentparams"
)

type KubernetesAgentOutput struct {
	components.JSONImporter

	AgentInstallName        string `json:"agentInstallName"`
	AgentInstallNameWindows string `json:"agentInstallNameWindows"`
}

// DockerAgent is a Docker installer on a remote Host
type KubernetesAgent struct {
	pulumi.ResourceState
	components.Component

	AgentInstallName        pulumi.StringOutput `pulumi:"agentInstallName"`
	AgentInstallNameWindows pulumi.StringOutput `pulumi:"agentInstallNameWindows"`
}

func (h *KubernetesAgent) Export(ctx *pulumi.Context, out *KubernetesAgentOutput) error {
	return components.Export(ctx, h, out)
}

func NewKubernetesAgent(e config.CommonEnvironment, clusterName string, kubeProvider *kubernetes.Provider, options ...kubernetesagentparams.Option) (*KubernetesAgent, error) {
	return components.NewComponent(e, clusterName, func(comp *KubernetesAgent) error {
		params, err := kubernetesagentparams.NewParams(&e, options...)
		if err != nil {
			return err
		}
		helmComponent, err := NewHelmInstallation(e, HelmInstallationArgs{
			KubeProvider: kubeProvider,
			Namespace:    params.Namespace,
			ValuesYAML:   params.HelmValues,
			Fakeintake:   params.FakeIntake,
		}, params.PulumiDependsOn...)
		if err != nil {
			return err
		}

		// Fill component
		comp.AgentInstallName = helmComponent.LinuxHelmReleaseName.Elem()
		if params.DeployWindows {
			comp.AgentInstallNameWindows = helmComponent.WindowsHelmReleaseName.Elem()
		}
		return nil
	})
}
