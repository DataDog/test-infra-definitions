package agent

import (
	"fmt"

	"golang.org/x/exp/maps"
	"gopkg.in/yaml.v3"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
	"github.com/DataDog/test-infra-definitions/resources/helm"

	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	kubeHelm "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi-random/sdk/v4/go/random"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	DatadogHelmRepo = "https://helm.datadoghq.com"
)

// HelmInstallationArgs is the set of arguments for creating a new HelmInstallation component
type HelmInstallationArgs struct {
	// KubeProvider is the Kubernetes provider to use
	KubeProvider *kubernetes.Provider
	// Namespace is the namespace in which to install the agent
	Namespace string
	// ValuesYAML is used to provide installation-specific values
	ValuesYAML pulumi.AssetOrArchiveArray
	// Fakeintake is used to configure the agent to send data to a fake intake
	Fakeintake *fakeintake.Fakeintake
	// DeployWindows is used to deploy the Windows agent
	DeployWindows bool
	// AgentFullImagePath is used to specify the full image path for the agent
	AgentFullImagePath string
	// ClusterAgentFullImagePath is used to specify the full image path for the cluster agent
	ClusterAgentFullImagePath string
	// DisableLogsContainerCollectAll is used to disable the collection of logs from all containers by default
	DisableLogsContainerCollectAll bool
	// DisableDualShipping is used to disable dual-shipping
	DisableDualShipping bool
	// OTelAgent is used to deploy the OTel agent instead of the classic agent
	OTelAgent bool
	// OTelConfig is used to provide a custom OTel configuration
	OTelConfig string
}

type HelmComponent struct {
	pulumi.ResourceState

	LinuxHelmReleaseName   pulumi.StringPtrOutput
	LinuxHelmReleaseStatus kubeHelm.ReleaseStatusOutput

	WindowsHelmReleaseName   pulumi.StringPtrOutput
	WindowsHelmReleaseStatus kubeHelm.ReleaseStatusOutput
}

func NewHelmInstallation(e config.Env, args HelmInstallationArgs, opts ...pulumi.ResourceOption) (*HelmComponent, error) {
	apiKey := e.AgentAPIKey()
	appKey := e.AgentAPPKey()
	baseName := "dda"
	opts = append(opts, pulumi.Providers(args.KubeProvider), e.WithProviders(config.ProviderRandom), pulumi.DeletedWith(args.KubeProvider))

	helmComponent := &HelmComponent{}
	if err := e.Ctx().RegisterComponentResource("dd:agent", "dda", helmComponent, opts...); err != nil {
		return nil, err
	}
	opts = append(opts, pulumi.Parent(helmComponent))

	// Create fixed cluster agent token
	randomClusterAgentToken, err := random.NewRandomString(e.Ctx(), "datadog-cluster-agent-token", &random.RandomStringArgs{
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
	ns, err := corev1.NewNamespace(e.Ctx(), args.Namespace, &corev1.NamespaceArgs{
		Metadata: metav1.ObjectMetaArgs{
			Name: pulumi.String(args.Namespace),
		},
	}, opts...)
	if err != nil {
		return nil, err
	}
	opts = append(opts, utils.PulumiDependsOn(ns))

	// Create secret if necessary
	secret, err := corev1.NewSecret(e.Ctx(), "datadog-credentials", &corev1.SecretArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: ns.Metadata.Name(),
			Name:      pulumi.Sprintf("%s-datadog-credentials", baseName),
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
	agentImagePath := dockerAgentFullImagePath(e, "", "", args.OTelAgent)
	if args.AgentFullImagePath != "" {
		agentImagePath = args.AgentFullImagePath
	}
	agentImagePath, agentImageTag := utils.ParseImageReference(agentImagePath)

	clusterAgentImagePath := dockerClusterAgentFullImagePath(e, "")
	if args.ClusterAgentFullImagePath != "" {
		clusterAgentImagePath = args.ClusterAgentFullImagePath
	}
	clusterAgentImagePath, clusterAgentImageTag := utils.ParseImageReference(clusterAgentImagePath)

	linuxInstallName := baseName + "-linux"

	values := buildLinuxHelmValues(baseName, agentImagePath, agentImageTag, clusterAgentImagePath, clusterAgentImageTag, randomClusterAgentToken.Result, !args.DisableLogsContainerCollectAll)
	values.configureImagePullSecret(imgPullSecret)
	values.configureFakeintake(e, args.Fakeintake, !args.DisableDualShipping)

	defaultYAMLValues := values.toYAMLPulumiAssetOutput()

	var valuesYAML pulumi.AssetOrArchiveArray
	valuesYAML = append(valuesYAML, defaultYAMLValues)
	valuesYAML = append(valuesYAML, args.ValuesYAML...)
	if args.OTelAgent {
		valuesYAML = append(valuesYAML, buildOTelConfigWithFakeintake(args.OTelConfig, args.Fakeintake))
	}

	linux, err := helm.NewInstallation(e, helm.InstallArgs{
		RepoURL:     DatadogHelmRepo,
		ChartName:   "datadog",
		InstallName: linuxInstallName,
		Namespace:   args.Namespace,
		ValuesYAML:  valuesYAML,
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
		values := buildWindowsHelmValues(baseName, agentImagePath, agentImageTag, clusterAgentImagePath, clusterAgentImageTag)
		values.configureImagePullSecret(imgPullSecret)
		values.configureFakeintake(e, args.Fakeintake, !args.DisableDualShipping)
		defaultYAMLValues := values.toYAMLPulumiAssetOutput()

		var windowsValuesYAML pulumi.AssetOrArchiveArray
		windowsValuesYAML = append(windowsValuesYAML, defaultYAMLValues)
		windowsValuesYAML = append(windowsValuesYAML, args.ValuesYAML...)

		windowsInstallName := baseName + "-windows"
		windows, err := helm.NewInstallation(e, helm.InstallArgs{
			RepoURL:     DatadogHelmRepo,
			ChartName:   "datadog",
			InstallName: windowsInstallName,
			Namespace:   args.Namespace,
			ValuesYAML:  windowsValuesYAML,
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

	if err := e.Ctx().RegisterResourceOutputs(helmComponent, resourceOutputs); err != nil {
		return nil, err
	}

	return helmComponent, nil
}

type HelmValues pulumi.Map

func buildLinuxHelmValues(baseName, agentImagePath, agentImageTag, clusterAgentImagePath, clusterAgentImageTag string, clusterAgentToken pulumi.StringInput, logsContainerCollectAll bool) HelmValues {
	return HelmValues{
		"datadog": pulumi.Map{
			"apiKeyExistingSecret": pulumi.String(baseName + "-datadog-credentials"),
			"appKeyExistingSecret": pulumi.String(baseName + "-datadog-credentials"),
			"checksCardinality":    pulumi.String("high"),
			"namespaceLabelsAsTags": pulumi.Map{
				"related_team": pulumi.String("team"),
			},
			"namespaceAnnotationsAsTags": pulumi.Map{
				"related_email": pulumi.String("email"),
			},
			"logs": pulumi.Map{
				"enabled":             pulumi.Bool(true),
				"containerCollectAll": pulumi.Bool(logsContainerCollectAll),
			},
			"dogstatsd": pulumi.Map{
				"originDetection": pulumi.Bool(true),
				"tagCardinality":  pulumi.String("high"),
				"useHostPort":     pulumi.Bool(true),
			},
			"apm": pulumi.Map{
				"portEnabled": pulumi.Bool(true),
				"instrumentation": pulumi.Map{
					"enabled": pulumi.Bool(true),
					"enabledNamespaces": pulumi.Array{
						pulumi.String("workload-mutated-lib-injection"),
					},
					"language_detection": pulumi.Map{
						"enabled": pulumi.Bool(true),
					},
				},
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
			"sbom": pulumi.Map{
				"host": pulumi.Map{
					"enabled": pulumi.Bool(true),
				},
				"containerImage": pulumi.Map{
					"enabled":                   pulumi.Bool(true),
					"uncompressedLayersSupport": pulumi.Bool(true),
				},
			},
			// The fake intake keeps payloads only for a hardcoded period of 15 minutes.
			// https://github.com/DataDog/datadog-agent/blob/34922393ce47261da9835d7bf62fb5e090e5fa55/test/fakeintake/server/server.go#L81
			// So, we need `container_image` and `sbom` checks to resubmit their payloads more frequently than that.
			"confd": pulumi.StringMap{
				"container_image.yaml": pulumi.String(utils.JSONMustMarshal(map[string]interface{}{
					"ad_identifiers": []string{"_container_image"},
					"init_config":    map[string]interface{}{},
					"instances": []map[string]interface{}{
						{
							"periodic_refresh_seconds": 300, // To have at least one refresh per test
						},
					},
				})),
				"sbom.yaml": pulumi.String(utils.JSONMustMarshal(map[string]interface{}{
					"ad_identifiers": []string{"_sbom"},
					"init_config":    map[string]interface{}{},
					"instances": []map[string]interface{}{
						{
							"periodic_refresh_seconds": 300, // To have at least one refresh per test
						},
					},
				})),
			},
			"env": pulumi.StringMapArray{
				pulumi.StringMap{
					"name":  pulumi.String("DD_EC2_METADATA_TIMEOUT"),
					"value": pulumi.String("5000"), // Unit is ms
				},
				pulumi.StringMap{
					"name":  pulumi.String("DD_TELEMETRY_ENABLED"),
					"value": pulumi.String("true"),
				},
				pulumi.StringMap{
					"name":  pulumi.String("DD_TELEMETRY_CHECKS"),
					"value": pulumi.String("*"),
				},
			},
		},
		"agents": pulumi.Map{
			"image": pulumi.Map{
				"repository":    pulumi.String(agentImagePath),
				"tag":           pulumi.String(agentImageTag),
				"doNotCheckTag": pulumi.Bool(true),
			},
			"priorityClassCreate": pulumi.Bool(true),
			"podAnnotations": pulumi.StringMap{
				"ad.datadoghq.com/agent.checks": pulumi.String(utils.JSONMustMarshal(
					map[string]interface{}{
						"openmetrics": map[string]interface{}{
							"init_config": map[string]interface{}{},
							"instances": []map[string]interface{}{
								{
									"openmetrics_endpoint": "http://localhost:6000/telemetry",
									"namespace":            "datadog.agent",
									"metrics": []string{
										".*",
									},
								},
							},
						},
					}),
				),
			},
			"containers": pulumi.Map{
				"agent": pulumi.Map{
					"resources": pulumi.StringMapMap{
						"requests": pulumi.StringMap{
							"cpu":    pulumi.String("400m"),
							"memory": pulumi.String("500Mi"),
						},
						"limits": pulumi.StringMap{
							"cpu":    pulumi.String("1000m"),
							"memory": pulumi.String("700Mi"),
						},
					},
				},
				"processAgent": pulumi.Map{
					"resources": pulumi.StringMapMap{
						"requests": pulumi.StringMap{
							"cpu":    pulumi.String("50m"),
							"memory": pulumi.String("150Mi"),
						},
						"limits": pulumi.StringMap{
							"cpu":    pulumi.String("200m"),
							"memory": pulumi.String("200Mi"),
						},
					},
				},
				"traceAgent": pulumi.Map{
					"resources": pulumi.StringMapMap{
						"requests": pulumi.StringMap{
							"cpu":    pulumi.String("10m"),
							"memory": pulumi.String("120Mi"),
						},
						"limits": pulumi.StringMap{
							"cpu":    pulumi.String("200m"),
							"memory": pulumi.String("200Mi"),
						},
					},
				},
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
			"resources": pulumi.StringMapMap{
				"requests": pulumi.StringMap{
					"cpu":    pulumi.String("50m"),
					"memory": pulumi.String("150Mi"),
				},
				"limits": pulumi.StringMap{
					"cpu":    pulumi.String("200m"),
					"memory": pulumi.String("200Mi"),
				},
			},
			"env": pulumi.StringMapArray{
				pulumi.StringMap{
					"name":  pulumi.String("DD_EC2_METADATA_TIMEOUT"),
					"value": pulumi.String("5000"), // Unit is ms
				},
				// This option is disabled by default and not exposed in the
				// Helm chart yet, so we need to set the env.
				pulumi.StringMap{
					"name":  pulumi.String("DD_ADMISSION_CONTROLLER_AUTO_INSTRUMENTATION_INJECT_AUTO_DETECTED_LIBRARIES"),
					"value": pulumi.String("true"),
				},
			},
		},
		"clusterChecksRunner": pulumi.Map{
			"enabled": pulumi.Bool(true),
			"image": pulumi.Map{
				"repository":    pulumi.String(agentImagePath),
				"tag":           pulumi.String(agentImageTag),
				"doNotCheckTag": pulumi.Bool(true),
			},
			"resources": pulumi.StringMapMap{
				"requests": pulumi.StringMap{
					"cpu":    pulumi.String("20m"),
					"memory": pulumi.String("300Mi"),
				},
				"limits": pulumi.StringMap{
					"cpu":    pulumi.String("200m"),
					"memory": pulumi.String("400Mi"),
				},
			},
		},
	}
}

func buildWindowsHelmValues(baseName string, agentImagePath, agentImageTag, _, _ string) HelmValues {
	return HelmValues{
		"targetSystem": pulumi.String("windows"),
		"datadog": pulumi.Map{
			"apiKeyExistingSecret": pulumi.String(baseName + "-datadog-credentials"),
			"appKeyExistingSecret": pulumi.String(baseName + "-datadog-credentials"),
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
			"serviceName":          pulumi.String(baseName + "-linux-datadog-cluster-agent"),
			"tokenSecretName":      pulumi.String(baseName + "-linux-datadog-cluster-agent"),
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

func (values HelmValues) configureFakeintake(e config.Env, fakeintake *fakeintake.Fakeintake, dualShipping bool) {
	if fakeintake == nil {
		return
	}

	var endpointsEnvVar pulumi.StringMapArray
	if dualShipping {
		if fakeintake.Scheme != "https" {
			e.Ctx().Log.Warn("Fakeintake is used in HTTP with dual-shipping, some endpoints will not work", nil)
		}
		endpointsEnvVar = pulumi.StringMapArray{
			pulumi.StringMap{
				"name":  pulumi.String("DD_SKIP_SSL_VALIDATION"),
				"value": pulumi.String("true"),
			},
			pulumi.StringMap{
				"name":  pulumi.String("DD_REMOTE_CONFIGURATION_NO_TLS_VALIDATION"),
				"value": pulumi.String("true"),
			},
			pulumi.StringMap{
				"name":  pulumi.String("DD_ADDITIONAL_ENDPOINTS"),
				"value": pulumi.Sprintf(`{"%s": ["FAKEAPIKEY"]}`, fakeintake.URL),
			},
			pulumi.StringMap{
				"name":  pulumi.String("DD_PROCESS_ADDITIONAL_ENDPOINTS"),
				"value": pulumi.Sprintf(`{"%s": ["FAKEAPIKEY"]}`, fakeintake.URL),
			},
			pulumi.StringMap{
				"name":  pulumi.String("DD_ORCHESTRATOR_EXPLORER_ORCHESTRATOR_ADDITIONAL_ENDPOINTS"),
				"value": pulumi.Sprintf(`{"%s": ["FAKEAPIKEY"]}`, fakeintake.URL),
			},
			pulumi.StringMap{
				"name":  pulumi.String("DD_LOGS_CONFIG_ADDITIONAL_ENDPOINTS"),
				"value": pulumi.Sprintf(`[{"host": "%s", "port": %v, "use_ssl": %t}]`, fakeintake.Host, fakeintake.Port, fakeintake.Scheme == "https"),
			},
			pulumi.StringMap{
				"name":  pulumi.String("DD_LOGS_CONFIG_USE_HTTP"),
				"value": pulumi.String("true"),
			},
			pulumi.StringMap{
				"name":  pulumi.String("DD_CONTAINER_IMAGE_ADDITIONAL_ENDPOINTS"),
				"value": pulumi.Sprintf(`[{"host": "%s"}]`, fakeintake.Host),
			},
			pulumi.StringMap{
				"name":  pulumi.String("DD_CONTAINER_LIFECYCLE_ADDITIONAL_ENDPOINTS"),
				"value": pulumi.Sprintf(`[{"host": "%s"}]`, fakeintake.Host),
			},
			pulumi.StringMap{
				"name":  pulumi.String("DD_SBOM_ADDITIONAL_ENDPOINTS"),
				"value": pulumi.Sprintf(`[{"host": "%s"}]`, fakeintake.Host),
			},
		}
	} else {
		endpointsEnvVar = pulumi.StringMapArray{
			pulumi.StringMap{
				"name":  pulumi.String("DD_DD_URL"),
				"value": pulumi.Sprintf("%s", fakeintake.URL),
			},
			pulumi.StringMap{
				"name":  pulumi.String("DD_PROCESS_CONFIG_PROCESS_DD_URL"),
				"value": pulumi.Sprintf("%s", fakeintake.URL),
			},
			pulumi.StringMap{
				"name":  pulumi.String("DD_APM_DD_URL"),
				"value": pulumi.Sprintf("%s", fakeintake.URL),
			},
			pulumi.StringMap{
				"name":  pulumi.String("DD_SKIP_SSL_VALIDATION"),
				"value": pulumi.String("true"),
			},
			pulumi.StringMap{
				"name":  pulumi.String("DD_REMOTE_CONFIGURATION_NO_TLS_VALIDATION"),
				"value": pulumi.String("true"),
			},
			pulumi.StringMap{
				"name":  pulumi.String("DD_LOGS_CONFIG_USE_HTTP"),
				"value": pulumi.String("true"),
			},
		}
	}

	for _, section := range []string{"datadog", "clusterAgent", "clusterChecksRunner"} {
		if _, found := values[section].(pulumi.Map)["env"]; !found {
			values[section].(pulumi.Map)["env"] = endpointsEnvVar
		} else {
			values[section].(pulumi.Map)["env"] = append(values[section].(pulumi.Map)["env"].(pulumi.StringMapArray), endpointsEnvVar...)
		}
	}
}

func (values HelmValues) toYAMLPulumiAssetOutput() pulumi.AssetOutput {
	return pulumi.Map(values).ToMapOutput().ApplyT(func(v map[string]interface{}) (pulumi.Asset, error) {
		yamlValues, err := yaml.Marshal(v)
		if err != nil {
			return nil, err
		}
		return pulumi.NewStringAsset(string(yamlValues)), nil
	}).(pulumi.AssetOutput)

}

func buildOTelConfigWithFakeintake(otelConfig string, fakeintake *fakeintake.Fakeintake) pulumi.AssetOutput {

	return fakeintake.URL.ApplyT(func(url string) (pulumi.Asset, error) {
		defaultConfig := map[string]interface{}{
			"exporters": map[string]interface{}{
				"datadog": map[string]interface{}{
					"metrics": map[string]interface{}{
						"endpoint": url,
					},
					"traces": map[string]interface{}{
						"endpoint": url,
					},
					"logs": map[string]interface{}{
						"endpoint": url,
					},
				},
			},
		}
		config := map[string]interface{}{}
		if err := yaml.Unmarshal([]byte(otelConfig), &config); err != nil {
			return nil, err
		}
		mergedConfig := utils.MergeMaps(config, defaultConfig)
		mergedConfigYAML, err := yaml.Marshal(mergedConfig)
		if err != nil {
			return nil, err
		}
		otelConfigValues := fmt.Sprintf(`
datadog:
  otelCollector:
    config: |
%s
`, utils.IndentMultilineString(string(mergedConfigYAML), 6))
		return pulumi.NewStringAsset(otelConfigValues), nil

	}).(pulumi.AssetOutput)
}
