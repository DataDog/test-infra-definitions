package dda

import (
	"encoding/json"
	"fmt"
	"strings"

	"dario.cat/mergo"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apiextensions"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"gopkg.in/yaml.v3"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
	componentskube "github.com/DataDog/test-infra-definitions/components/kubernetes"
)

func K8sAppDefinition(e config.Env, kubeProvider *kubernetes.Provider, namespace string, fakeIntake *fakeintake.Fakeintake, kubeletTLSVerify bool, clusterName string, customDda string, opts ...pulumi.ResourceOption) (*componentskube.Workload, *componentskube.KubernetesObjectRef, error) {
	apiKey := e.AgentAPIKey()
	appKey := e.AgentAPPKey()
	baseName := "dda-with-operator"
	opts = append(opts, pulumi.Provider(kubeProvider), pulumi.Parent(kubeProvider), pulumi.DeletedWith(kubeProvider))

	k8sComponent := &componentskube.Workload{}
	if err := e.Ctx().RegisterComponentResource("dd:agent-with-operator", "dda", k8sComponent, opts...); err != nil {
		return nil, nil, err
	}

	opts = append(opts, pulumi.Parent(k8sComponent))

	ns, err := corev1.NewNamespace(
		e.Ctx(),
		namespace,
		&corev1.NamespaceArgs{
			Metadata: metav1.ObjectMetaArgs{
				Name: pulumi.String(namespace),
			},
		},
		opts...,
	)
	if err != nil {
		return nil, nil, err
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
		return nil, nil, err
	}
	opts = append(opts, utils.PulumiDependsOn(secret))

	ddaConfig := buildDDAConfig(baseName, clusterName, kubeletTLSVerify)
	if fakeIntake != nil {
		configureFakeIntake(ddaConfig, fakeIntake)
	}
	ddaConfig, err = mergeYamlToConfig(ddaConfig, customDda)

	if err != nil {
		return nil, nil, err
	}

	// Image pull secrets need to be configured after custom DDA config merge because pulumi.StringOutput cannot be marshalled to JSON
	var imagePullSecret *corev1.Secret
	if e.ImagePullRegistry() != "" {
		imagePullSecret, err = utils.NewImagePullSecret(e, namespace, opts...)
		if err != nil {
			return nil, nil, err
		}
		opts = append(opts, utils.PulumiDependsOn(imagePullSecret))
		configureImagePullSecret(ddaConfig, imagePullSecret)
	}

	ddaName := "datadog-agent"
	if e.PipelineID() != "" {
		ddaName = strings.Join([]string{ddaName, e.PipelineID()}, "-")
	}

	_, err = apiextensions.NewCustomResource(e.Ctx(), "datadog-agent", &apiextensions.CustomResourceArgs{
		ApiVersion: pulumi.String("datadoghq.com/v2alpha1"),
		Kind:       pulumi.String("DatadogAgent"),
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(ddaName),
			Namespace: pulumi.String(namespace),
		},
		OtherFields: ddaConfig,
	}, opts...)
	if err != nil {
		return nil, nil, err
	}

	ddaRef, err := componentskube.NewKubernetesObjRef(e, baseName, namespace, "DatadogAgent", pulumi.String("").ToStringOutput(), pulumi.String("datadoghq.com/v2alpha1").ToStringOutput(), map[string]string{"app": baseName})

	return k8sComponent, ddaRef, nil
}

func buildDDAConfig(baseName string, clusterName string, kubeletTLSVerify bool) kubernetes.UntypedArgs {
	return kubernetes.UntypedArgs{
		"spec": pulumi.Map{
			"global": pulumi.Map{
				"clusterName": pulumi.String(clusterName),
				"kubelet": pulumi.Map{
					"tlsVerify": pulumi.Bool(kubeletTLSVerify),
				},
				"credentials": pulumi.Map{
					"apiSecret": pulumi.Map{
						"secretName": pulumi.String(baseName + "-datadog-credentials"),
						"keyName":    pulumi.String("api-key"),
					},
					"appSecret": pulumi.Map{
						"secretName": pulumi.String(baseName + "-datadog-credentials"),
						"keyName":    pulumi.String("app-key"),
					},
				},
			},
			"features": pulumi.Map{
				"clusterChecks": pulumi.Map{
					"enabled":                 pulumi.Bool(true),
					"useClusterChecksRunners": pulumi.Bool(true),
				},
				"dogstatsd": pulumi.Map{
					"tagCardinality": pulumi.String("high"),
				},
				"logCollection": pulumi.Map{
					"enabled":                    pulumi.Bool(true),
					"containerCollectAll":        pulumi.Bool(true),
					"containerCollectUsingFiles": pulumi.Bool(true),
				},
				"prometheusScrape": pulumi.Map{
					"enabled": pulumi.Bool(true),
					"version": pulumi.Int(2),
				},
				"liveProcessCollection": pulumi.Map{
					"enabled": pulumi.Bool(true),
				},
				"eventCollection": pulumi.Map{
					"collectKubernetesEvents": pulumi.Bool(false),
				},
			},
		},
	}
}

func configureFakeIntake(config kubernetes.UntypedArgs, fakeintake *fakeintake.Fakeintake) {
	if fakeintake == nil {
		return
	}
	endpointsEnvVar := pulumi.StringMapArray{
		pulumi.StringMap{
			"name":  pulumi.String("DD_DD_URL"),
			"value": pulumi.String(fmt.Sprintf("%v", fakeintake.URL)),
		},
		pulumi.StringMap{
			"name":  pulumi.String("DD_PROCESS_CONFIG_PROCESS_DD_URL"),
			"value": pulumi.String(fmt.Sprintf("%v", fakeintake.URL)),
		},
		pulumi.StringMap{
			"name":  pulumi.String("DD_APM_DD_URL"),
			"value": pulumi.String(fmt.Sprintf("%v", fakeintake.URL)),
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
	for _, section := range []string{"nodeAgent", "clusterAgent", "clusterChecksRunner"} {
		if _, found := config["spec"].(pulumi.Map)["override"]; !found {
			config["spec"].(pulumi.Map)["override"] = pulumi.Map{
				section: pulumi.Map{
					"env": endpointsEnvVar,
				},
			}
		} else if _, found = config["spec"].(pulumi.Map)["override"].(pulumi.Map)[section]; !found {
			config["spec"].(pulumi.Map)["override"].(pulumi.Map)[section] = pulumi.Map{
				"env": endpointsEnvVar,
			}
		} else if _, found = config["spec"].(pulumi.Map)["override"].(pulumi.Map)[section].(pulumi.Map)["env"]; !found {
			config["spec"].(pulumi.Map)["override"].(pulumi.Map)[section].(pulumi.Map)["env"] = endpointsEnvVar
		} else {
			config["spec"].(pulumi.Map)["override"].(pulumi.Map)[section].(pulumi.Map)["env"] = append(config["spec"].(pulumi.Map)["override"].(pulumi.Map)[section].(pulumi.Map)["env"].(pulumi.StringMapArray), endpointsEnvVar...)
		}
	}
}

func configureImagePullSecret(config kubernetes.UntypedArgs, secret *corev1.Secret) {
	if secret == nil {
		return
	}

	for _, section := range []string{"nodeAgent", "clusterAgent", "clusterChecksRunner"} {
		if _, found := config["spec"].(map[string]interface{})["override"].(map[string]interface{})[section]; !found {
			config["spec"].(map[string]interface{})["override"].(map[string]interface{})[section] = pulumi.Map{
				"image": pulumi.Map{
					"pullSecrets": pulumi.MapArray{
						pulumi.Map{
							"name": secret.Metadata.Name(),
						},
					},
				},
			}
		} else if _, found = config["spec"].(map[string]interface{})["override"].(map[string]interface{})[section].(map[string]interface{})["image"]; !found {
			config["spec"].(map[string]interface{})["override"].(map[string]interface{})[section].(map[string]interface{})["image"] = pulumi.Map{
				"pullSecrets": pulumi.MapArray{
					pulumi.Map{
						"name": secret.Metadata.Name(),
					},
				},
			}
		} else {
			config["spec"].(map[string]interface{})["override"].(map[string]interface{})[section].(map[string]interface{})["image"].(map[string]interface{})["pullSecrets"] = pulumi.MapArray{
				pulumi.Map{
					"name": secret.Metadata.Name(),
				},
			}
		}
	}
}

func mergeYamlToConfig(config kubernetes.UntypedArgs, yamlConfig string) (kubernetes.UntypedArgs, error) {
	var configMap, yamlMap map[string]interface{}
	configJSON, err := json.Marshal(config)
	if err != nil {
		fmt.Println(fmt.Sprintf("Error marshalling original DDA config: %v)", err))
		return config, err
	}

	if err := json.Unmarshal(configJSON, &configMap); err != nil {
		return config, fmt.Errorf("error unmarshalling original DDA config: %v", err)
	}
	if err := yaml.Unmarshal([]byte(yamlConfig), &yamlMap); err != nil {
		return config, fmt.Errorf("error unmarshalling new DDA yaml config: %v", err)
	}

	if err := mergo.Map(&configMap, yamlMap, mergo.WithOverride); err != nil {
		return config, fmt.Errorf("error merging DDA configs: %v", err)
	}

	merged, err := json.Marshal(configMap)
	if err != nil {
		return config, fmt.Errorf("error marshalling merged DDA config: %v", err)
	}

	var mergedConfig kubernetes.UntypedArgs
	if err = json.Unmarshal(merged, &mergedConfig); err != nil {
		return config, fmt.Errorf("error ummarshalling merged DDA config: %v", err)
	}

	return mergedConfig, nil
}
