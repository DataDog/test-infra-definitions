package gke

import (
	"github.com/DataDog/test-infra-definitions/common"
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components"
	kubeComp "github.com/DataDog/test-infra-definitions/components/kubernetes"
	"github.com/DataDog/test-infra-definitions/resources/gcp"
	"github.com/DataDog/test-infra-definitions/resources/gcp/gke"

	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Params struct {
}

type Option = func(*Params) error

func NewParams(options ...Option) (*Params, error) {
	params := &Params{}
	return common.ApplyOption(params, options)
}

func NewGKECluster(env gcp.Environment, opts ...Option) (*kubeComp.Cluster, error) {
	_, err := NewParams(opts...)
	if err != nil {
		return nil, err
	}

	return components.NewComponent(&env, env.Namer.ResourceName("gke"), func(comp *kubeComp.Cluster) error {
		cluster, kubeConfig, err := gke.NewCluster(env, "gke")
		if err != nil {
			return err
		}

		comp.ClusterName = cluster.Name
		comp.KubeConfig = kubeConfig

		// Building Kubernetes provider
		gkeKubeProvider, err := kubernetes.NewProvider(env.Ctx(), env.Namer.ResourceName("k8s-provider"), &kubernetes.ProviderArgs{
			EnableServerSideApply: pulumi.BoolPtr(true),
			Kubeconfig:            utils.KubeConfigYAMLToJSON(kubeConfig),
		}, env.WithProviders(config.ProviderGCP))
		if err != nil {
			return err
		}
		comp.KubeProvider = gkeKubeProvider

		return nil
	})
}
