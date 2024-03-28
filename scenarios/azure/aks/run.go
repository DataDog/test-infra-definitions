package aks

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components"
	"github.com/DataDog/test-infra-definitions/components/datadog/agent"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/cpustress"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/dogstatsd"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/mutatedbyadmissioncontroller"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/nginx"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/prometheus"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/redis"
	dogstatsdstandalone "github.com/DataDog/test-infra-definitions/components/datadog/dogstatsd-standalone"
	kubeComp "github.com/DataDog/test-infra-definitions/components/kubernetes"
	"github.com/DataDog/test-infra-definitions/resources/azure"
	"github.com/DataDog/test-infra-definitions/resources/azure/aks"

	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Run(ctx *pulumi.Context) error {
	env, err := azure.NewEnvironment(ctx)
	if err != nil {
		return err
	}

	clusterComp, err := components.NewComponent(*env.CommonEnvironment, env.Namer.ResourceName("aks"), func(comp *kubeComp.Cluster) error {
		cluster, kubeConfig, err := aks.NewCluster(env, "aks", nil)
		if err != nil {
			return err
		}

		// Filling Kubernetes component from EKS cluster
		comp.ClusterName = cluster.Name
		comp.KubeConfig = kubeConfig

		// Building Kubernetes provider
		aksKubeProvider, err := kubernetes.NewProvider(env.Ctx, env.Namer.ResourceName("k8s-provider"), &kubernetes.ProviderArgs{
			EnableServerSideApply: pulumi.BoolPtr(true),
			Kubeconfig:            utils.KubeConfigYAMLToJSON(kubeConfig),
		}, env.WithProviders(config.ProviderAzure))
		if err != nil {
			return err
		}
		comp.KubeProvider = aksKubeProvider

		var dependsOnCrd pulumi.ResourceOption
		// TODO: add node pools and fake intake without using ecs

		// Deploy the agent
		if env.AgentDeploy() {
			customValues := `
datadog:
  kubelet:
    host:
      valueFrom:
        fieldRef:
          fieldPath: spec.nodeName
    hostCAPath: /etc/kubernetes/certs/kubeletserver.crt
providers:
  aks:
    enabled: true
`

			helmComponent, err := agent.NewHelmInstallation(*env.CommonEnvironment, agent.HelmInstallationArgs{
				KubeProvider: aksKubeProvider,
				Namespace:    "datadog",
				ValuesYAML:   pulumi.AssetOrArchiveArray{pulumi.NewStringAsset(customValues)},
			}, nil)
			if err != nil {
				return err
			}

			ctx.Export("agent-linux-helm-install-name", helmComponent.LinuxHelmReleaseName)
			ctx.Export("agent-linux-helm-install-status", helmComponent.LinuxHelmReleaseStatus)

			dependsOnCrd = utils.PulumiDependsOn(helmComponent)
		}

		// Deploy standalone dogstatsd
		if env.DogstatsdDeploy() {
			if _, err := dogstatsdstandalone.K8sAppDefinition(*env.CommonEnvironment, aksKubeProvider, "dogstatsd-standalone", nil, true, ""); err != nil {
				return err
			}
		}

		// Deploy testing workload
		if env.TestingWorkloadDeploy() {
			if _, err := nginx.K8sAppDefinition(*env.CommonEnvironment, aksKubeProvider, "workload-nginx", dependsOnCrd); err != nil {
				return err
			}

			if _, err := redis.K8sAppDefinition(*env.CommonEnvironment, aksKubeProvider, "workload-redis", dependsOnCrd); err != nil {
				return err
			}

			if _, err := cpustress.K8sAppDefinition(*env.CommonEnvironment, aksKubeProvider, "workload-cpustress"); err != nil {
				return err
			}

			// dogstatsd clients that report to the Agent
			if _, err := dogstatsd.K8sAppDefinition(*env.CommonEnvironment, aksKubeProvider, "workload-dogstatsd", 8125, "/var/run/datadog/dsd.socket"); err != nil {
				return err
			}

			// dogstatsd clients that report to the dogstatsd standalone deployment
			if _, err := dogstatsd.K8sAppDefinition(*env.CommonEnvironment, aksKubeProvider, "workload-dogstatsd-standalone", dogstatsdstandalone.HostPort, dogstatsdstandalone.Socket); err != nil {
				return err
			}

			if _, err := prometheus.K8sAppDefinition(*env.CommonEnvironment, aksKubeProvider, "workload-prometheus"); err != nil {
				return err
			}

			if _, err := mutatedbyadmissioncontroller.K8sAppDefinition(*env.CommonEnvironment, aksKubeProvider, "workload-mutated"); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	return clusterComp.Export(ctx, nil)
}
