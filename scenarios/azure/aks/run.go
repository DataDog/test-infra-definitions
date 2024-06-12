package aks

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components"
	"github.com/DataDog/test-infra-definitions/components/datadog/agent/helm"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/cpustress"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/dogstatsd"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/mutatedbyadmissioncontroller"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/nginx"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/prometheus"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/redis"
	dogstatsdstandalone "github.com/DataDog/test-infra-definitions/components/datadog/dogstatsd-standalone"
	"github.com/DataDog/test-infra-definitions/components/datadog/kubernetesagentparams"
	kubeComp "github.com/DataDog/test-infra-definitions/components/kubernetes"
	"github.com/DataDog/test-infra-definitions/resources/azure"
	"github.com/DataDog/test-infra-definitions/resources/azure/aks"

	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const kataRuntimeClass = "kata-mshv-vm-isolation"

func Run(ctx *pulumi.Context) error {
	env, err := azure.NewEnvironment(ctx)
	if err != nil {
		return err
	}

	clusterComp, err := components.NewComponent(&env, env.Namer.ResourceName("aks"), func(comp *kubeComp.Cluster) error {
		cluster, kubeConfig, err := aks.NewCluster(env, "aks", nil)
		if err != nil {
			return err
		}

		// Filling Kubernetes component from EKS cluster
		comp.ClusterName = cluster.Name
		comp.KubeConfig = kubeConfig

		// Building Kubernetes provider
		aksKubeProvider, err := kubernetes.NewProvider(env.Ctx(), env.Namer.ResourceName("k8s-provider"), &kubernetes.ProviderArgs{
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
			// On Kata nodes, AKS uses the node-name (like aks-kata-21213134-vmss000000) as the only SAN in the Kubelet
			// certificate. However, the DNS name aks-kata-21213134-vmss000000 is not resolvable, so it cannot be used
			// to reach the Kubelet. Thus we need to use `tlsVerify: false` and `and `status.hostIP` as `host` in
			// the Helm values
			customValues := `
datadog:
  kubelet:
    host:
      valueFrom:
        fieldRef:
          fieldPath: status.hostIP
    hostCAPath: /etc/kubernetes/certs/kubeletserver.crt
    tlsVerify: false
providers:
  aks:
    enabled: true
`
			k8sAgentOptions := make([]kubernetesagentparams.Option, 0)
			k8sAgentOptions = append(
				k8sAgentOptions,
				kubernetesagentparams.WithNamespace("datadog"),
				kubernetesagentparams.WithHelmValues(customValues),
			)

			k8sAgentComponent, err := helm.NewKubernetesAgent(&env, env.Namer.ResourceName("datadog-agent"), aksKubeProvider, k8sAgentOptions...)

			if err != nil {
				return err
			}

			dependsOnCrd = utils.PulumiDependsOn(k8sAgentComponent)
		}

		// Deploy standalone dogstatsd
		if env.DogstatsdDeploy() {
			if _, err := dogstatsdstandalone.K8sAppDefinition(&env, aksKubeProvider, "dogstatsd-standalone", nil, true, ""); err != nil {
				return err
			}
		}

		// Deploy testing workload
		if env.TestingWorkloadDeploy() {
			if _, err := nginx.K8sAppDefinition(&env, aksKubeProvider, "workload-nginx", "", true, dependsOnCrd); err != nil {
				return err
			}

			if _, err := nginx.K8sAppDefinition(&env, aksKubeProvider, "workload-nginx-kata", kataRuntimeClass, true, dependsOnCrd); err != nil {
				return err
			}

			if _, err := redis.K8sAppDefinition(&env, aksKubeProvider, "workload-redis", true, dependsOnCrd); err != nil {
				return err
			}

			if _, err := cpustress.K8sAppDefinition(&env, aksKubeProvider, "workload-cpustress"); err != nil {
				return err
			}

			// dogstatsd clients that report to the Agent
			if _, err := dogstatsd.K8sAppDefinition(&env, aksKubeProvider, "workload-dogstatsd", 8125, "/var/run/datadog/dsd.socket"); err != nil {
				return err
			}

			// dogstatsd clients that report to the dogstatsd standalone deployment
			if _, err := dogstatsd.K8sAppDefinition(&env, aksKubeProvider, "workload-dogstatsd-standalone", dogstatsdstandalone.HostPort, dogstatsdstandalone.Socket); err != nil {
				return err
			}

			if _, err := prometheus.K8sAppDefinition(&env, aksKubeProvider, "workload-prometheus"); err != nil {
				return err
			}

			if _, err := mutatedbyadmissioncontroller.K8sAppDefinition(&env, aksKubeProvider, "workload-mutated", "workload-mutated-lib-injection"); err != nil {
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
