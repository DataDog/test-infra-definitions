package helm

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

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
		pulumiResourceOptions := append(params.PulumiResourceOptions, pulumi.Parent(comp))

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
		}, pulumiResourceOptions...)
		if err != nil {
			return err
		}

		platform := "linux"
		appVersion := helmComponent.LinuxHelmReleaseStatus.AppVersion().Elem()
		version := helmComponent.LinuxHelmReleaseStatus.Version().Elem()

		baseName := "dda-" + platform

		comp.LinuxNodeAgent, err = agent.NewKubernetesObjRef(e, baseName+"-nodeAgent", params.Namespace, "Pod", appVersion, version, map[string]string{
			"app": baseName + "-datadog",
		})

		if err != nil {
			return err
		}

		comp.LinuxClusterAgent, err = agent.NewKubernetesObjRef(e, baseName+"-clusterAgent", params.Namespace, "Pod", appVersion, version, map[string]string{
			"app": baseName + "-datadog-cluster-agent",
		})

		if err != nil {
			return err
		}

		comp.LinuxClusterChecks, err = agent.NewKubernetesObjRef(e, baseName+"-clusterChecks", params.Namespace, "Pod", appVersion, version, map[string]string{
			"app": baseName + "-datadog-clusterchecks",
		})

		if params.DeployWindows {
			platform = "windows"
			appVersion = helmComponent.WindowsHelmReleaseStatus.AppVersion().Elem()
			version = helmComponent.WindowsHelmReleaseStatus.Version().Elem()

			baseName = "dda-" + platform

			comp.WindowsNodeAgent, err = agent.NewKubernetesObjRef(e, baseName+"-nodeAgent", params.Namespace, "Pod", appVersion, version, map[string]string{
				"app": baseName + "-datadog",
			})
			if err != nil {
				return err
			}

			comp.WindowsClusterAgent, err = agent.NewKubernetesObjRef(e, baseName+"-clusterAgent", params.Namespace, "Pod", appVersion, version, map[string]string{
				"app": baseName + "-datadog-cluster-agent",
			})
			if err != nil {
				return err
			}

			comp.WindowsClusterChecks, err = agent.NewKubernetesObjRef(e, baseName+"-clusterChecks", params.Namespace, "Pod", appVersion, version, map[string]string{
				"app": baseName + "-datadog-clusterchecks",
			})
			if err != nil {
				return err
			}
		}

		if err != nil {
			return err
		}

		return nil
	})
}
