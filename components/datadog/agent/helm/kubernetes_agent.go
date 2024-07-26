package helm

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"

	"github.com/DataDog/test-infra-definitions/components/datadog/agent"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/components"
	"github.com/DataDog/test-infra-definitions/components/datadog/kubernetesagentparams"
)

func NewKubernetesAgent(e config.Env, resourceName string, kubeProvider *kubernetes.Provider, options ...kubernetesagentparams.Option) (*agent.KubernetesAgent, error) {
	return components.NewComponent(e, resourceName, func(comp *agent.KubernetesAgent) error {
		params, err := kubernetesagentparams.NewParams(e, options...)
		if err != nil {
			return err
		}

		helmComponent, err := agent.NewHelmInstallation(e, agent.HelmInstallationArgs{
			KubeProvider:                   kubeProvider,
			DeployWindows:                  params.DeployWindows,
			Namespace:                      params.Namespace,
			ValuesYAML:                     params.HelmValues,
			Fakeintake:                     params.FakeIntake,
			AgentFullImagePath:             params.AgentFullImagePath,
			ClusterAgentFullImagePath:      params.ClusterAgentFullImagePath,
			DisableLogsContainerCollectAll: params.DisableLogsContainerCollectAll,
			OTelAgent:                      params.OTelAgent,
		}, params.PulumiResourceOptions...)
		if err != nil {
			return err
		}

		platform := "linux"
		appVersion := helmComponent.LinuxHelmReleaseStatus.AppVersion().Elem()
		version := helmComponent.LinuxHelmReleaseStatus.Version().Elem()

		if params.DeployWindows {
			platform = "windows"
			appVersion = helmComponent.WindowsHelmReleaseStatus.AppVersion().Elem()
			version = helmComponent.WindowsHelmReleaseStatus.Version().Elem()
		}

		baseName := "dda-" + platform

		comp.NodeAgent, err = agent.NewKubernetesObjRef(e, baseName+"-nodeAgent", params.Namespace, "Pod", appVersion, version, map[string]string{
			"app": baseName + "-datadog",
		})

		if err != nil {
			return err
		}

		comp.ClusterAgent, err = agent.NewKubernetesObjRef(e, baseName+"-clusterAgent", params.Namespace, "Pod", appVersion, version, map[string]string{
			"app": baseName + "-datadog-cluster-agent",
		})

		if err != nil {
			return err
		}

		comp.ClusterChecks, err = agent.NewKubernetesObjRef(e, baseName+"-clusterChecks", params.Namespace, "Pod", appVersion, version, map[string]string{
			"app": baseName + "-datadog-clusterchecks",
		})

		if err != nil {
			return err
		}

		return nil
	})
}
