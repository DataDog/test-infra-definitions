package agent

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/helm"

	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	kubeHelm "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/helm/v3"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	DatadogHelmRepo = "https://helm.datadoghq.com"
)

func NewHelmInstallation(e config.CommonEnvironment, kubeProvider *kubernetes.Provider, namespace string, valuesFilepaths []string, opts ...pulumi.ResourceOption) (*kubeHelm.Release, error) {
	apiKey := e.AgentAPIKey()
	appKey := e.AgentAPPKey()

	// Create namespace if necessary
	ns, err := corev1.NewNamespace(e.Ctx, namespace, &corev1.NamespaceArgs{
		Metadata: metav1.ObjectMetaArgs{
			Name: pulumi.String(namespace),
		},
	}, pulumi.Provider(kubeProvider))
	if err != nil {
		return nil, err
	}

	// Create secret if necessary
	secret, err := corev1.NewSecret(e.Ctx, "dd-datadog-credentials", &corev1.SecretArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: ns.Metadata.Name(),
			Name:      pulumi.String("dd-datadog-credentials"),
		},
		StringData: pulumi.StringMap{
			"api-key": apiKey,
			"app-key": appKey,
		},
	}, pulumi.Provider(kubeProvider))
	if err != nil {
		return nil, err
	}

	// Compute some values
	installName := "dda"
	agentImagePath := DockerFullImagePath(&e, "")
	agentImagePath, agentImageTag := utils.ParseImageReference(agentImagePath)

	opts = append(opts, utils.PulumiDependsOn(ns, secret))
	return helm.NewInstallation(e, helm.InstallArgs{
		KubernetesProvider: kubeProvider,
		RepoURL:            DatadogHelmRepo,
		ChartName:          "datadog",
		InstallName:        installName,
		Namespace:          namespace,
		ValuesFilePaths:    valuesFilepaths,
		Values:             buildDefaultHelmValues(installName, agentImagePath, agentImageTag),
	}, opts...)
}

func buildDefaultHelmValues(installName string, agentImagePath, agentImageTag string) pulumi.Map {
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
			"metricsProvider": pulumi.Map{
				"enabled":           pulumi.Bool(true),
				"useDatadogMetrics": pulumi.Bool(true),
			},
		},
		"clusterChecksRunner": pulumi.Map{
			"enabled": pulumi.Bool(true),
		},
	}
}
