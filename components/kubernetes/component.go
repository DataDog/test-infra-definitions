package kubernetes

import (
	"github.com/DataDog/test-infra-definitions/components"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// The type that is used to import the KubernetesCluster component
type ClusterOutput struct {
	components.JSONImporter

	ClusterName string `json:"clusterName"`
	KubeConfig  string `json:"kubeConfig"`
}

// Cluster represents a Kubernetes cluster
type Cluster struct {
	pulumi.ResourceState
	components.Component

	KubeProvider pulumi.ProviderResource

	ClusterName pulumi.StringOutput `pulumi:"clusterName"`
	KubeConfig  pulumi.StringOutput `pulumi:"kubeConfig"`
}

func (c *Cluster) Export(ctx *pulumi.Context, out *ClusterOutput) error {
	return components.Export(ctx, c, out)
}

type WorkloadComponent struct {
	pulumi.ResourceState
	components.Component
}
