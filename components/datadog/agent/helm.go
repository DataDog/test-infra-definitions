package agent

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/resources/helm"

	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	kubeHelm "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/helm/v3"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	DatadogHelmRepo = "https://helm.datadoghq.com"
)

type HelmInstallationArgs struct {
	KubeProvider *kubernetes.Provider
	Namespace    string
	ValuesYAML   pulumi.AssetOrArchiveArrayInput
}

func NewHelmInstallation(e config.CommonEnvironment, args HelmInstallationArgs, opts ...pulumi.ResourceOption) (*kubeHelm.Release, error) {
	apiKey := e.AgentAPIKey()
	appKey := e.AgentAPPKey()
	installName := "dda"
	opts = append(opts, pulumi.Provider(args.KubeProvider), pulumi.Parent(args.KubeProvider), pulumi.DeletedWith(args.KubeProvider))

	// Create namespace if necessary
	ns, err := corev1.NewNamespace(e.Ctx, args.Namespace, &corev1.NamespaceArgs{
		Metadata: metav1.ObjectMetaArgs{
			Name: pulumi.String(args.Namespace),
		},
	}, opts...)
	if err != nil {
		return nil, err
	}

	// Create secret if necessary
	secret, err := corev1.NewSecret(e.Ctx, "datadog-credentials", &corev1.SecretArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: ns.Metadata.Name(),
			Name:      pulumi.String(fmt.Sprintf("%s-datadog-credentials", installName)),
		},
		StringData: pulumi.StringMap{
			"api-key": apiKey,
			"app-key": appKey,
		},
	}, opts...)
	if err != nil {
		return nil, err
	}

	// Compute some values
	agentImagePath := DockerAgentFullImagePath(&e, "")
	agentImagePath, agentImageTag := utils.ParseImageReference(agentImagePath)

	clusterAgentImagePath := DockerClusterAgentFullImagePath(&e, "")
	clusterAgentImagePath, clusterAgentImageTag := utils.ParseImageReference(clusterAgentImagePath)

	opts = append(opts, utils.PulumiDependsOn(ns, secret))
	return helm.NewInstallation(e, helm.InstallArgs{
		RepoURL:     DatadogHelmRepo,
		ChartName:   "datadog",
		InstallName: installName,
		Namespace:   args.Namespace,
		ValuesYAML:  args.ValuesYAML,
		Values:      buildDefaultHelmValues(installName, agentImagePath, agentImageTag, clusterAgentImagePath, clusterAgentImageTag),
	}, opts...)
}

func buildDefaultHelmValues(installName string, agentImagePath, agentImageTag, clusterAgentImagePath, clusterAgentImageTag string) pulumi.Map {
	return pulumi.Map{
		"datadog": pulumi.Map{
			"apiKeyExistingSecret": pulumi.String(installName + "-datadog-credentials"),
			"appKeyExistingSecret": pulumi.String(installName + "-datadog-credentials"),
			"checksCardinality":    pulumi.String("high"),
			"logs": pulumi.Map{
				"enabled":             pulumi.Bool(true),
				"containerCollectAll": pulumi.Bool(true),
			},
			"dogstatsd": pulumi.Map{
				"originDetection": pulumi.Bool(true),
				"tagCardinality":  pulumi.String("high"),
				"useHostPort":     pulumi.Bool(true),
			},
			"apm": pulumi.Map{
				"portEnabled": pulumi.Bool(true),
			},
			"processAgent": pulumi.Map{
				"processCollection": pulumi.Bool(true),
			},
			"helmCheck": pulumi.Map{
				"enabled": pulumi.Bool(true),
			},
		},
		"agents": pulumi.Map{
			"image": pulumi.Map{
				"repository":    pulumi.String(agentImagePath),
				"tag":           pulumi.String(agentImageTag),
				"doNotCheckTag": pulumi.Bool(true),
			},
		},
		"clusterAgent": pulumi.Map{
			"enabled": pulumi.Bool(true),
			"image": pulumi.Map{
				"repository":    pulumi.String(clusterAgentImagePath),
				"tag":           pulumi.String(clusterAgentImageTag),
				"doNotCheckTag": pulumi.Bool(true),
			},
			"metricsProvider": pulumi.Map{
				"enabled":           pulumi.Bool(true),
				"useDatadogMetrics": pulumi.Bool(true),
			},
		},
		"clusterChecksRunner": pulumi.Map{
			"enabled": pulumi.Bool(true),
			"image": pulumi.Map{
				"repository":    pulumi.String(agentImagePath),
				"tag":           pulumi.String(agentImageTag),
				"doNotCheckTag": pulumi.Bool(true),
			},
		},
	}
}
