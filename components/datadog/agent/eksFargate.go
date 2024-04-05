package agent

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"

	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	v1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/rbac/v1"
	"github.com/pulumi/pulumi-random/sdk/v4/go/random"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func EKSFargateHelmInstallation(e config.CommonEnvironment, kubeProvider *kubernetes.Provider, namespace string, fakeIntakeParam *fakeintake.Fakeintake, opts ...pulumi.ResourceOption) (*HelmComponent, error) {
	opts = append(opts, pulumi.Providers(kubeProvider), e.WithProviders(config.ProviderRandom), pulumi.Parent(kubeProvider), pulumi.DeletedWith(kubeProvider))

	clusterRoleArgs := v1.ClusterRoleArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("datadog-agent"),
			Namespace: pulumi.String("fargate"),
		},
		Rules: v1.PolicyRuleArray{
			v1.PolicyRuleArgs{
				ApiGroups: pulumi.StringArray{
					pulumi.String(""),
				},
				Resources: pulumi.StringArray{
					pulumi.String("nodes"),
					pulumi.String("namespaces"),
					pulumi.String("endpoints"),
				},
				Verbs: pulumi.StringArray{
					pulumi.String("get"),
					pulumi.String("list"),
				},
			},
			v1.PolicyRuleArgs{
				ApiGroups: pulumi.StringArray{
					pulumi.String(""),
				},
				Resources: pulumi.StringArray{
					pulumi.String("nodes/metrics"),
					pulumi.String("nodes/spec"),
					pulumi.String("nodes/stats"),
					pulumi.String("nodes/proxy"),
					pulumi.String("nodes/pods"),
					pulumi.String("nodes/healthz"),
				},
				Verbs: pulumi.StringArray{
					pulumi.String("get"),
				},
			},
		},
	}

	clusterRoleBindingArgs := v1.ClusterRoleBindingArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name: pulumi.String("datadog-agent"),
		},
		RoleRef: v1.RoleRefArgs{
			ApiGroup: pulumi.String("rbac.authorization.k8s.io"),
			Kind:     pulumi.String("ClusterRole"),
			Name:     pulumi.String("datadog-agent"),
		},
		Subjects: v1.SubjectArray{
			&v1.SubjectArgs{
				Kind:      pulumi.String("ServiceAccount"),
				Name:      pulumi.String("datadog-agent"),
				Namespace: pulumi.String(namespace),
			},
		},
	}

	if _, err := v1.NewClusterRole(e.Ctx, "datadog-agent", &clusterRoleArgs, opts...); err != nil {
		return nil, err
	}

	if _, err := v1.NewClusterRoleBinding(e.Ctx, "datadog-agent", &clusterRoleBindingArgs, opts...); err != nil {
		return nil, err
	}

	randomClusterAgentToken, err := random.NewRandomString(e.Ctx, "datadog-cluster-agent-token", &random.RandomStringArgs{
		Lower:   pulumi.Bool(true),
		Upper:   pulumi.Bool(true),
		Length:  pulumi.Int(32),
		Numeric: pulumi.Bool(false),
		Special: pulumi.Bool(false),
	}, opts...)
	if err != nil {
		return nil, err
	}

	apiKey := e.AgentAPIKey()
	appKey := e.AgentAPPKey()
	secret_dca, err := corev1.NewSecret(e.Ctx, "datadog-credentials-dca", &corev1.SecretArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: pulumi.StringPtr("datadog-agent"),
			Name:      pulumi.Sprintf("datadog-secret"),
		},
		StringData: pulumi.StringMap{
			"api-key": apiKey,
			"app-key": appKey,
			"token":   randomClusterAgentToken.Result,
		},
	}, opts...)
	if err != nil {
		return nil, err
	}

	secret_agent_inject, err := corev1.NewSecret(e.Ctx, "datadog-credentials-injection", &corev1.SecretArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: pulumi.StringPtr("fargate"),
			Name:      pulumi.Sprintf("datadog-secret"),
		},
		StringData: pulumi.StringMap{
			"api-key": apiKey,
			"app-key": appKey,
			"token":   randomClusterAgentToken.Result,
		},
	}, opts...)
	if err != nil {
		return nil, err
	}

	opts = append(opts, utils.PulumiDependsOn(secret_dca), utils.PulumiDependsOn(secret_agent_inject))

	customValues := `
datadog:
  kubelet:
    tlsVerify: false
agents:
  useHostNetwork: true
  enabled: false
clusterAgent:
  tokenExistingSecret: datadog-secret
  enabled: true
  admissionController: 
    enabled: true
    agentSidecarInjection:
      enabled: true
      provider: fargate
`

	helmComponent, err := NewHelmInstallation(e, HelmInstallationArgs{
		KubeProvider: kubeProvider,
		Namespace:    "datadog-agent",
		ValuesYAML: pulumi.AssetOrArchiveArray{
			pulumi.NewStringAsset(customValues),
		},
		Fakeintake:        fakeIntakeParam,
		ClusterAgentToken: randomClusterAgentToken,
	}, nil)

	if err != nil {
		return nil, err
	}

	serviceAccountArgs := corev1.ServiceAccountArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("datadog-agent"),
			Namespace: pulumi.String(namespace),
		},
	}

	if _, err := corev1.NewServiceAccount(e.Ctx, "datadog-agent", &serviceAccountArgs, opts...); err != nil {
		return nil, err
	}

	return helmComponent, nil
}
