package agent

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/DataDog/test-infra-definitions/components"
)

type KubernetesAgentOutput struct {
	components.JSONImporter

	NodeAgent     KubernetesObjRefOutput `json:"nodeAgent"`
	ClusterAgent  KubernetesObjRefOutput `json:"clusterAgent"`
	ClusterChecks KubernetesObjRefOutput `json:"clusterChecks"`
}

// KubernetesAgent is an installer to install the Datadog Agent on a Kubernetes cluster.
type KubernetesAgent struct {
	pulumi.ResourceState
	components.Component

	NodeAgent     *KubernetesObjectRef `pulumi:"nodeAgent"`
	ClusterAgent  *KubernetesObjectRef `pulumi:"clusterAgent"`
	ClusterChecks *KubernetesObjectRef `pulumi:"clusterChecks"`
}

func (h *KubernetesAgent) Export(ctx *pulumi.Context, out *KubernetesAgentOutput) error {
	return components.Export(ctx, h, out)
}

//    agent-linux-helm-install-name  : "dda-linux"
//    agent-linux-helm-install-status: {
//        appVersion: "7"
//        chart     : "datadog"
//        name      : "dda-linux"
//        namespace : "datadog"
//        revision  : 1
//        status    : "deployed"
//        version   : "3.66.0"
//    }
