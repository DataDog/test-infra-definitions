package dda

import (
	"fmt"

	"dario.cat/mergo"
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/datadog/agentwithoperatorparams"
	componentskube "github.com/DataDog/test-infra-definitions/components/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/yaml"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	baseName = "agent-with-operator"
)

type datadogAgentWorkload struct {
	ctx             *pulumi.Context
	opts            *agentwithoperatorparams.Params
	name            string
	clusterName     string
	imagePullSecret *corev1.Secret
}

func K8sAppDefinition(e config.Env, kubeProvider *kubernetes.Provider, ddaOpts []agentwithoperatorparams.Option, opts ...pulumi.ResourceOption) (*componentskube.Workload, error) {
	if ddaOpts == nil {
		return nil, nil
	}
	apiKey := e.AgentAPIKey()
	appKey := e.AgentAPPKey()
	clusterName := e.Ctx().Stack()

	ddaOptions, err := agentwithoperatorparams.NewParams(ddaOpts...)
	if err != nil {
		return nil, err
	}

	opts = append(opts, pulumi.Provider(kubeProvider), pulumi.Parent(kubeProvider), pulumi.DeletedWith(kubeProvider))

	k8sComponent := &componentskube.Workload{}
	if err := e.Ctx().RegisterComponentResource("dd:agent-with-operator", "dda", k8sComponent, opts...); err != nil {
		return nil, err
	}

	opts = append(opts, pulumi.Parent(k8sComponent))

	// Create datadog-credentials secret if necessary
	secret, err := corev1.NewSecret(e.Ctx(), "datadog-credentials", &corev1.SecretArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: pulumi.String(ddaOptions.Namespace),
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

	// Create imagePullSecret
	var imagePullSecret *corev1.Secret
	if e.ImagePullRegistry() != "" {
		imagePullSecret, err = utils.NewImagePullSecret(e, ddaOptions.Namespace, opts...)
		if err != nil {
			return nil, err
		}
		opts = append(opts, utils.PulumiDependsOn(imagePullSecret))
	}

	ddaWorkload := datadogAgentWorkload{
		ctx:             e.Ctx(),
		opts:            ddaOptions,
		name:            ddaOptions.DDAConfig.Name,
		clusterName:     clusterName,
		imagePullSecret: imagePullSecret,
	}

	if err = ddaWorkload.buildDDAConfig(opts...); err != nil {
		return nil, err
	}

	return k8sComponent, nil
}

func (d datadogAgentWorkload) buildDDAConfig(opts ...pulumi.ResourceOption) error {
	ctx := d.ctx
	defaultYamlTransformations := d.defaultDDAYamlTransformations()

	if d.opts.DDAConfig.YamlFilePath != "" {
		_, err := yaml.NewConfigGroup(ctx, d.name, &yaml.ConfigGroupArgs{
			Files:           []string{d.opts.DDAConfig.YamlFilePath},
			Transformations: defaultYamlTransformations,
		}, opts...)

		if err != nil {
			return err
		}
	} else if d.opts.DDAConfig.YamlConfig != "" {
		_, err := yaml.NewConfigGroup(ctx, d.name, &yaml.ConfigGroupArgs{
			YAML:            []string{d.opts.DDAConfig.YamlConfig},
			Transformations: defaultYamlTransformations,
		}, opts...)

		if err != nil {
			return err
		}
	} else if d.opts.DDAConfig.MapConfig != nil {
		_, err := yaml.NewConfigGroup(ctx, d.name, &yaml.ConfigGroupArgs{
			Objs:            []map[string]interface{}{d.opts.DDAConfig.MapConfig},
			Transformations: defaultYamlTransformations,
		}, opts...)

		if err != nil {
			return err
		}
	} else {
		_, err := yaml.NewConfigGroup(ctx, d.name, &yaml.ConfigGroupArgs{
			Objs:            []map[string]interface{}{d.defaultDDAConfig()},
			Transformations: d.defaultDDAYamlTransformations(),
		}, opts...)

		if err != nil {
			return err
		}

	}
	return nil
}

func (d datadogAgentWorkload) defaultDDAConfig() map[string]interface{} {
	return map[string]interface{}{
		"apiVersion": "datadoghq.com/v2alpha1",
		"kind":       "DatadogAgent",
		"metadata": map[string]interface{}{
			"name":      d.opts.DDAConfig.Name,
			"namespace": d.opts.Namespace,
		},
		"spec": map[string]interface{}{
			"global": map[string]interface{}{
				"clusterName": d.clusterName,
				"kubelet": map[string]interface{}{
					"tlsVerify": d.opts.KubeletTLSVerify,
				},
				"credentials": map[string]interface{}{
					"apiSecret": map[string]interface{}{
						"secretName": baseName + "-datadog-credentials",
						"keyName":    "api-key",
					},
					"appSecret": map[string]interface{}{
						"secretName": baseName + "-datadog-credentials",
						"keyName":    "app-key",
					},
				},
			},
			"features": map[string]interface{}{
				"clusterChecks": map[string]interface{}{
					"enabled":                 true,
					"useClusterChecksRunners": true,
				},
			},
		},
	}
}

func (d datadogAgentWorkload) fakeIntakeEnvVars() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"name":  "DD_DD_URL",
			"value": d.opts.FakeIntake.URL,
		},
		{
			"name":  "DD_PROCESS_CONFIG_PROCESS_DD_URL",
			"value": d.opts.FakeIntake.URL,
		},
		{
			"name":  "DD_APM_DD_URL",
			"value": d.opts.FakeIntake.URL,
		},
		{
			"name":  "DD_SKIP_SSL_VALIDATION",
			"value": "true",
		},
		{
			"name":  "DD_REMOTE_CONFIGURATION_NO_TLS_VALIDATION",
			"value": "true",
		},
		{
			"name":  "DD_LOGS_CONFIG_USE_HTTP",
			"value": "true",
		},
	}
}

func (d datadogAgentWorkload) defaultDDAYamlTransformations() []yaml.Transformation {
	return []yaml.Transformation{
		// Override custom DDAConfig with required defaults
		func(state map[string]interface{}, opts ...pulumi.ResourceOption) {
			defaultDDAConfig := d.defaultDDAConfig()
			err := mergo.Merge(&state, defaultDDAConfig)
			if err != nil {
				d.ctx.Log.Debug(fmt.Sprintf("There was a problem merging the default DDA config: %v", err), nil)
			}
		},
		// Configure metadata
		func(state map[string]interface{}, opts ...pulumi.ResourceOption) {
			if state["metadata"] == nil {
				state["metadata"] = map[string]interface{}{
					"name":      d.opts.DDAConfig.Name,
					"namespace": d.opts.Namespace,
				}
			}
			state["metadata"].(map[string]interface{})["namespace"] = d.opts.Namespace

			state["metadata"].(map[string]interface{})["name"] = d.opts.DDAConfig.Name

			if state["spec"].(map[string]interface{})["global"] == nil {
				state["spec"].(map[string]interface{})["global"] = map[string]interface{}{
					"clusterName": d.clusterName,
				}
			} else {
				state["spec"].(map[string]interface{})["global"].(map[string]interface{})["clusterName"] = d.clusterName
			}
		},
		// Configure global
		func(state map[string]interface{}, opts ...pulumi.ResourceOption) {
			defaultGlobal := map[string]interface{}{
				"clusterName": d.clusterName,
				"kubelet": map[string]interface{}{
					"tlsVerify": d.opts.KubeletTLSVerify,
				},
				"credentials": map[string]interface{}{
					"apiSecret": map[string]interface{}{
						"secretName": baseName + "-datadog-credentials",
						"keyName":    "api-key",
					},
					"appSecret": map[string]interface{}{
						"secretName": baseName + "-datadog-credentials",
						"keyName":    "app-key",
					},
				},
			}
			if state["spec"].(map[string]interface{})["global"] == nil {
				state["spec"].(map[string]interface{})["global"] = defaultGlobal
			} else {
				stateGlobal := state["spec"].(map[string]interface{})["global"].(map[string]interface{})
				for k, v := range defaultGlobal {
					if stateGlobal[k] == nil {
						stateGlobal[k] = v
					} else {
						err := mergo.Map(stateGlobal[k], defaultGlobal[k])
						if err != nil {
							d.ctx.Log.Debug(fmt.Sprintf("Error merging YAML maps: %v", err), nil)
						}
					}
				}
			}
		},
		// Configure Fake Intake
		func(state map[string]interface{}, opts ...pulumi.ResourceOption) {
			if d.opts.FakeIntake == nil {
				return
			}
			for _, section := range []string{"nodeAgent", "clusterAgent", "clusterChecksRunner"} {
				if state["spec"].(map[string]interface{})["override"] == nil {
					state["spec"].(map[string]interface{})["override"] = map[string]interface{}{
						section: map[string]interface{}{
							"env": d.fakeIntakeEnvVars(),
						},
					}
				}
				if state["spec"].(map[string]interface{})["override"].(map[string]interface{})[section] == nil {
					state["spec"].(map[string]interface{})["override"].(map[string]interface{})[section] = map[string]interface{}{
						"env": d.fakeIntakeEnvVars(),
					}
				}
				if state["spec"].(map[string]interface{})["override"].(map[string]interface{})[section].(map[string]interface{})["env"] == nil {
					state["spec"].(map[string]interface{})["override"].(map[string]interface{})[section].(map[string]interface{})["env"] = d.fakeIntakeEnvVars()
				}
				if state["spec"].(map[string]interface{})["override"].(map[string]interface{})[section].(map[string]interface{})["env"].([]map[string]interface{}) != nil {
					env := state["spec"].(map[string]interface{})["override"].(map[string]interface{})[section].(map[string]interface{})["env"].([]map[string]interface{})
					env = append(env, d.fakeIntakeEnvVars()...)
				}
			}
		},
		//	Configure Image pull secret
		func(state map[string]interface{}, opts ...pulumi.ResourceOption) {
			if d.imagePullSecret == nil {
				return
			}
			for _, section := range []string{"nodeAgent", "clusterAgent", "clusterChecksRunner"} {
				if _, found := state["spec"].(map[string]interface{})["override"].(map[string]interface{})[section]; !found {
					state["spec"].(map[string]interface{})["override"].(map[string]interface{})[section] = map[string]interface{}{
						"image": map[string]interface{}{
							"pullSecrets": map[string]interface{}{
								"name": d.imagePullSecret.Metadata.Name(),
							},
						},
					}
				}
				if _, found := state["spec"].(map[string]interface{})["override"].(map[string]interface{})[section].(map[string]interface{})["image"]; !found {
					state["spec"].(map[string]interface{})["override"].(map[string]interface{})[section].(map[string]interface{})["image"] = map[string]interface{}{
						"pullSecrets": []map[string]interface{}{{
							"name": d.imagePullSecret.Metadata.Name(),
						}},
					}
				}
				if _, found := state["spec"].(map[string]interface{})["override"].(map[string]interface{})[section].(map[string]interface{})["image"]; !found {
					state["spec"].(map[string]interface{})["override"].(map[string]interface{})[section].(map[string]interface{})["image"] = map[string]interface{}{
						"pullSecrets": []map[string]interface{}{{
							"name": d.imagePullSecret.Metadata.Name(),
						}},
					}
				}
				if _, found := state["spec"].(map[string]interface{})["override"].(map[string]interface{})[section].(map[string]interface{})["image"].(map[string]interface{})["pullSecrets"]; found {
					pullSecrets := state["spec"].(map[string]interface{})["override"].(map[string]interface{})[section].(map[string]interface{})["image"].(map[string]interface{})["pullSecrets"].([]map[string]interface{})
					pullSecrets = append(pullSecrets, map[string]interface{}{
						"name": d.imagePullSecret.Metadata.Name(),
					})
				}
			}
		},
	}
}
