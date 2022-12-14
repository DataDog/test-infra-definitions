package agent

import (
	"github.com/DataDog/test-infra-definitions/common/config"
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

var defaultAgentValues = pulumi.Map{
	"datadog": pulumi.Map{
		"apiKeyExistingSecret": pulumi.String("dd-datadog-credentials"),
		"appKeyExistingSecret": pulumi.String("dd-datadog-credentials"),
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

func NewHelmInstallation(e config.CommonEnvironment, kubeProvider *kubernetes.Provider, namespace string, valueFiles []string) (*kubeHelm.Release, error) {
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

	// Install Helm chart
	return helm.NewInstallation(e, kubeProvider, DatadogHelmRepo, "datadog", "dd", namespace, defaultAgentValues, valueFiles, pulumi.DependsOn([]pulumi.Resource{ns, secret}))
}
