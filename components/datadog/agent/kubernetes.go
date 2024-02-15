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

	InstallNameLinux   string `json:"installNameLinux"`
	InstallNameWindows string `json:"installNameWindows"`
}

// KubernetesAgent is an installer to install the Datadog Agent on a Kubernetes cluster.
type KubernetesAgent struct {
	pulumi.ResourceState
	components.Component

	InstallNameLinux   pulumi.StringOutput `pulumi:"installNameLinux"`
	InstallNameWindows pulumi.StringOutput `pulumi:"installNameWindows"`
}

func (h *KubernetesAgent) Export(ctx *pulumi.Context, out *KubernetesAgentOutput) error {
	return components.Export(ctx, h, out)
}

func NewKubernetesAgent(e config.CommonEnvironment, resourceName string, kubeProvider *kubernetes.Provider, options ...kubernetesagentparams.Option) (*KubernetesAgent, error) {
	return components.NewComponent(e, resourceName, func(comp *KubernetesAgent) error {
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
		comp.InstallNameLinux = helmComponent.LinuxHelmReleaseName.Elem()
		if params.DeployWindows {
			comp.InstallNameWindows = helmComponent.WindowsHelmReleaseName.Elem()
		}
		return nil
	})
}
