package agent

import (
	"golang.org/x/exp/maps"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	ddfakeintake "github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
	"github.com/DataDog/test-infra-definitions/resources/helm"

	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	kubeHelm "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/helm/v3"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi-random/sdk/v4/go/random"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	DatadogHelmRepo = "https://helm.datadoghq.com"
)

type HelmInstallationArgs struct {
	KubeProvider  *kubernetes.Provider
	Namespace     string
	ValuesYAML    pulumi.AssetOrArchiveArrayInput
	Fakeintake    *ddfakeintake.ConnectionExporter
	DeployWindows bool
}

type HelmComponent struct {
	pulumi.ResourceState

	LinuxHelmReleaseName   pulumi.StringPtrOutput
	LinuxHelmReleaseStatus kubeHelm.ReleaseStatusOutput

	WindowsHelmReleaseName   pulumi.StringPtrOutput
	WindowsHelmReleaseStatus kubeHelm.ReleaseStatusOutput
}

func NewHelmInstallation(e config.CommonEnvironment, args HelmInstallationArgs, opts ...pulumi.ResourceOption) (*HelmComponent, error) {
	apiKey := e.AgentAPIKey()
	appKey := e.AgentAPPKey()
	installName := "dda"
	opts = append(opts, pulumi.Providers(args.KubeProvider), e.WithProvider(config.ProviderRandom), pulumi.Parent(args.KubeProvider), pulumi.DeletedWith(args.KubeProvider))

	helmComponent := &HelmComponent{}
	if err := e.Ctx.RegisterComponentResource("dd:agent", "dda", helmComponent, opts...); err != nil {
		return nil, err
	}
	opts = append(opts, pulumi.Parent(helmComponent))

	// Create fixed cluster agent token
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
	secret, err := corev1.NewSecret(e.Ctx, "datadog-credentials", &corev1.SecretArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: ns.Metadata.Name(),
			Name:      pulumi.Sprintf("%s-datadog-credentials", installName),
		},
		StringData: pulumi.StringMap{
			"api-key": apiKey,
			"app-key": appKey,
		},
	}, opts...)
	if err != nil {
		return nil, err
	}
	opts = append(opts, utils.PulumiDependsOn(secret))

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
	agentImagePath := DockerAgentFullImagePath(&e, "", "")
	agentImagePath, agentImageTag := utils.ParseImageReference(agentImagePath)

	clusterAgentImagePath := DockerClusterAgentFullImagePath(&e, "")
	clusterAgentImagePath, clusterAgentImageTag := utils.ParseImageReference(clusterAgentImagePath)

	linuxInstallName := installName
	if args.DeployWindows {
		linuxInstallName += "-linux"
	}

	values := buildLinuxHelmValues(installName, agentImagePath, agentImageTag, clusterAgentImagePath, clusterAgentImageTag, randomClusterAgentToken.Result)
	values.configureImagePullSecret(imgPullSecret)
	values.configureFakeintake(args.Fakeintake)

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
		values.configureFakeintake(args.Fakeintake)

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

type HelmValues pulumi.Map

func buildLinuxHelmValues(installName, agentImagePath, agentImageTag, clusterAgentImagePath, clusterAgentImageTag string, clusterAgentToken pulumi.StringInput) HelmValues {
	return HelmValues{
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
			"prometheusScrape": pulumi.Map{
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
			"token": clusterAgentToken,
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

func buildWindowsHelmValues(installName string, agentImagePath, agentImageTag, _, _ string) HelmValues {
	return HelmValues{
		"targetSystem": pulumi.String("windows"),
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
			"prometheusScrape": pulumi.Map{
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
		// Make the Windows node agents target the Linux cluster agent
		"clusterAgent": pulumi.Map{
			"enabled": pulumi.Bool(false),
		},
		"existingClusterAgent": pulumi.Map{
			"join":                 pulumi.Bool(true),
			"serviceName":          pulumi.String(installName + "-linux-datadog-cluster-agent"),
			"tokenSecretName":      pulumi.String(installName + "-linux-datadog-cluster-agent"),
			"clusterchecksEnabled": pulumi.Bool(false),
		},
		"clusterChecksRunner": pulumi.Map{
			"enabled": pulumi.Bool(false),
		},
	}
}

func (values HelmValues) configureImagePullSecret(secret *corev1.Secret) {
	if secret == nil {
		return
	}

	for _, section := range []string{"agents", "clusterAgent", "clusterChecksRunner"} {
		if _, found := values[section].(pulumi.Map)["image"]; found {
			values[section].(pulumi.Map)["image"].(pulumi.Map)["pullSecrets"] = pulumi.MapArray{
				pulumi.Map{
					"name": secret.Metadata.Name(),
				},
			}
		}
	}
}

func (values HelmValues) configureFakeintake(fakeintake *ddfakeintake.ConnectionExporter) {
	if fakeintake == nil {
		return
	}

	additionalEndpointsEnvVar := pulumi.MapArray{
		pulumi.Map{
			"name":  pulumi.String("DD_SKIP_SSL_VALIDATION"),
			"value": pulumi.String("true"),
		},
		pulumi.Map{
			"name":  pulumi.String("DD_ADDITIONAL_ENDPOINTS"),
			"value": pulumi.Sprintf(`{"https://%s": ["FAKEAPIKEY"]}`, fakeintake.Host),
		},
		pulumi.Map{
			"name":  pulumi.String("DD_PROCESS_ADDITIONAL_ENDPOINTS"),
			"value": pulumi.Sprintf(`{"http://%s": ["FAKEAPIKEY"]}`, fakeintake.Host),
		},
		pulumi.Map{
			"name":  pulumi.String("DD_ORCHESTRATOR_EXPLORER_ORCHESTRATOR_ADDITIONAL_ENDPOINTS"),
			"value": pulumi.Sprintf(`{"http://%s": ["FAKEAPIKEY"]}`, fakeintake.Host),
		},
		pulumi.Map{
			"name":  pulumi.String("DD_LOGS_CONFIG_ADDITIONAL_ENDPOINTS"),
			"value": pulumi.Sprintf(`[{"host": "%s"}]`, fakeintake.Host),
		},
		pulumi.Map{
			"name":  pulumi.String("DD_LOGS_CONFIG_USE_HTTP"),
			"value": pulumi.String("true"),
		},
	}

	for _, section := range []string{"datadog", "clusterAgent", "clusterChecksRunner"} {
		if _, found := values[section].(pulumi.Map)["env"]; !found {
			values[section].(pulumi.Map)["env"] = additionalEndpointsEnvVar
		} else {
			values[section].(pulumi.Map)["env"] = append(values[section].(pulumi.Map)["env"].(pulumi.MapArray), additionalEndpointsEnvVar...)
		}
	}
}
