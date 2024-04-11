package agent

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
	"github.com/DataDog/test-infra-definitions/resources/helm"
	"golang.org/x/exp/maps"

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

	helmComponent, err := NewHelmCAInstallation(e, HelmInstallationArgs{
		KubeProvider: kubeProvider,
		Namespace:    "datadog-agent",
		ValuesYAML: pulumi.AssetOrArchiveArray{
			pulumi.NewStringAsset(customValues),
		},
		Fakeintake: fakeIntakeParam,
	}, opts...)

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

func NewHelmCAInstallation(e config.CommonEnvironment, args HelmInstallationArgs, opts ...pulumi.ResourceOption) (*HelmComponent, error) {
	apiKey := e.AgentAPIKey()
	appKey := e.AgentAPPKey()
	installName := "ddca"
	opts = append(opts, pulumi.Providers(args.KubeProvider), e.WithProviders(config.ProviderRandom), pulumi.Parent(args.KubeProvider), pulumi.DeletedWith(args.KubeProvider))

	helmComponent := &HelmComponent{}
	if err := e.Ctx.RegisterComponentResource("dd:agent", "ddca", helmComponent, opts...); err != nil {
		return nil, err
	}
	opts = append(opts, pulumi.Parent(helmComponent))

	var randomClusterAgentToken *random.RandomString

	randomClusterAgentToken, err := random.NewRandomString(e.Ctx, "datadog-cluster-agent-inj-token", &random.RandomStringArgs{
		Lower:   pulumi.Bool(true),
		Upper:   pulumi.Bool(true),
		Length:  pulumi.Int(32),
		Numeric: pulumi.Bool(false),
		Special: pulumi.Bool(false),
	}, opts...)
	if err != nil {
		return nil, err
	}

	// Create fargate namespace if necessary
	fgns, err := corev1.NewNamespace(e.Ctx, "fargate", &corev1.NamespaceArgs{
		Metadata: metav1.ObjectMetaArgs{
			Name: pulumi.String("fargate"),
		},
	}, opts...)
	if err != nil {
		return nil, err
	}
	opts = append(opts, utils.PulumiDependsOn(fgns))

	// Create namespace if necessary
	ns, err := corev1.NewNamespace(e.Ctx, args.Namespace, &corev1.NamespaceArgs{
		Metadata: metav1.ObjectMetaArgs{
			Name: pulumi.String(args.Namespace),
		},
	}, opts...)
	if err != nil {
		return nil, err
	}
	opts = append(opts, utils.PulumiDependsOn(ns))

	// Create secret if necessary
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

	// Create image pull secret if necessary
	var imgPullSecret *corev1.Secret
	if e.ImagePullRegistry() != "" {
		imgPullSecret, err = NewImagePullSecret(e, args.Namespace, opts...)
		if err != nil {
			return nil, err
		}
		opts = append(opts, utils.PulumiDependsOn(imgPullSecret))
	}

	// Compute some values
	agentImagePath := dockerAgentFullImagePath(&e, "", "")
	if args.AgentFullImagePath != "" {
		agentImagePath = args.AgentFullImagePath
	}
	agentImagePath, agentImageTag := utils.ParseImageReference(agentImagePath)

	clusterAgentImagePath := dockerClusterAgentFullImagePath(&e, "")
	if args.ClusterAgentFullImagePath != "" {
		clusterAgentImagePath = args.ClusterAgentFullImagePath
	}
	clusterAgentImagePath, clusterAgentImageTag := utils.ParseImageReference(clusterAgentImagePath)

	linuxInstallName := installName
	if args.DeployWindows {
		linuxInstallName += "-linux"
	}

	values := buildLinuxHelmValues(installName, agentImagePath, agentImageTag, clusterAgentImagePath, clusterAgentImageTag, randomClusterAgentToken.Result)
	values.configureImagePullSecret(imgPullSecret)
	values.configureFakeintake(e, args.Fakeintake)

	linux, err := helm.NewInstallation(e, helm.InstallArgs{
		RepoURL:     DatadogHelmRepo,
		ChartName:   "datadog",
		InstallName: linuxInstallName,
		Namespace:   args.Namespace,
		ValuesYAML:  args.ValuesYAML,
		Values:      pulumi.Map(values),
	}, opts...)
	if err != nil {
		return nil, err
	}

	helmComponent.LinuxHelmReleaseName = linux.Name
	helmComponent.LinuxHelmReleaseStatus = linux.Status

	resourceOutputs := pulumi.Map{
		"linuxHelmReleaseName":   linux.Name,
		"linuxHelmReleaseStatus": linux.Status,
	}

	if args.DeployWindows {
		values := buildWindowsHelmValues(installName, agentImagePath, agentImageTag, clusterAgentImagePath, clusterAgentImageTag)
		values.configureImagePullSecret(imgPullSecret)
		values.configureFakeintake(e, args.Fakeintake)

		windows, err := helm.NewInstallation(e, helm.InstallArgs{
			RepoURL:     DatadogHelmRepo,
			ChartName:   "datadog",
			InstallName: installName + "-windows",
			Namespace:   args.Namespace,
			ValuesYAML:  args.ValuesYAML,
			Values:      pulumi.Map(values),
		}, opts...)
		if err != nil {
			return nil, err
		}

		helmComponent.WindowsHelmReleaseName = windows.Name
		helmComponent.WindowsHelmReleaseStatus = windows.Status

		maps.Copy(resourceOutputs, pulumi.Map{
			"windowsHelmReleaseName":   windows.Name,
			"windowsHelmReleaseStatus": windows.Status,
		})
	}

	if err := e.Ctx.RegisterResourceOutputs(helmComponent, resourceOutputs); err != nil {
		return nil, err
	}

	return helmComponent, nil
}
