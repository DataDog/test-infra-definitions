package agent

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/DataDog/test-infra-definitions/components"
)

type KubernetesAgentOutput struct {
	components.JSONImporter

	LinuxNodeAgent     KubernetesObjRefOutput `json:"linuxNodeAgent"`
	LinuxClusterAgent  KubernetesObjRefOutput `json:"linuxClusterAgent"`
	LinuxClusterChecks KubernetesObjRefOutput `json:"linuxClusterChecks"`

	WindowsNodeAgent     KubernetesObjRefOutput `json:"windowsNodeAgent"`
	WindowsClusterAgent  KubernetesObjRefOutput `json:"windowsClusterAgent"`
	WindowsClusterChecks KubernetesObjRefOutput `json:"windowsClusterChecks"`
}

// KubernetesAgent is an installer to install the Datadog Agent on a Kubernetes cluster.
type KubernetesAgent struct {
	pulumi.ResourceState
	components.Component

	LinuxNodeAgent     *KubernetesObjectRef `pulumi:"linuxNodeAgent"`
	LinuxClusterAgent  *KubernetesObjectRef `pulumi:"linuxClusterAgent"`
	LinuxClusterChecks *KubernetesObjectRef `pulumi:"linuxClusterChecks"`

	WindowsNodeAgent     *KubernetesObjectRef `pulumi:"windowsNodeAgent"`
	WindowsClusterAgent  *KubernetesObjectRef `pulumi:"windowsClusterAgent"`
	WindowsClusterChecks *KubernetesObjectRef `pulumi:"windowsClusterChecks"`
}

func (h *KubernetesAgent) Export(ctx *pulumi.Context, out *KubernetesAgentOutput) error {
	return components.Export(ctx, h, out)
}
