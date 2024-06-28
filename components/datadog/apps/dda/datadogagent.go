package dda

import (
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

func K8sAppDefinition(e config.Env, kubeProvider *kubernetes.Provider, namespace string, fakeIntake *fakeintake.Fakeintake, kubeletTLSVerify bool, clusterName string, customDda string, opts ...pulumi.ResourceOption) (*componentskube.Workload, error) {
	opts = append(opts, pulumi.Provider(kubeProvider), pulumi.Parent(kubeProvider), pulumi.DeletedWith(kubeProvider))

	k8sComponent := &componentskube.Workload{}
	if err := e.Ctx().RegisterComponentResource("dd:agent-with-operator", "dda", k8sComponent, opts...); err != nil {
		return nil, err
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
		return nil, err
	}

	opts = append(opts, utils.PulumiDependsOn(ns))

	var imagePullSecrets corev1.LocalObjectReferenceArray
	if e.ImagePullRegistry() != "" {
		imgPullSecret, err := utils.NewImagePullSecret(e, namespace, opts...)
		if err != nil {
			return nil, err
		}

		imagePullSecrets = append(imagePullSecrets, corev1.LocalObjectReferenceArgs{
			Name: imgPullSecret.Metadata.Name(),
		})
	}

	ddaArgs := kubernetes.UntypedArgs{}
	err = yaml.Unmarshal([]byte(customDda), &ddaArgs)
	if err != nil {
		return nil, err
	}

	defaultArgs := kubernetes.UntypedArgs{
		"spec": pulumi.Map{
			"global": pulumi.Map{
				"credentials": pulumi.Map{
					"apiKey": pulumi.StringInput(e.AgentAPIKey()),
					"appKey": pulumi.StringInput(e.AgentAPPKey()),
				},
				"clusterName": pulumi.String(clusterName),
				"kubelet": pulumi.Map{
					"tlsVerify": pulumi.Bool(kubeletTLSVerify),
				},
			},
		},
	}

	if e.AgentUseFakeintake() {
		err = mergo.Merge(&ddaArgs, kubernetes.UntypedArgs{
			"spec": pulumi.Map{
				"override": pulumi.Map{
					"nodeAgent": pulumi.Map{
						"env": pulumi.MapArray{
							pulumi.Map{
								"name":  pulumi.String("DD_ADDITIONAL_ENDPOINTS"),
								"value": pulumi.Sprintf(`{"%s": ["FAKEAPIKEY"]}`, fakeIntake.URL),
							},
						},
					},
					"clusterAgent": pulumi.Map{
						"env": pulumi.MapArray{
							pulumi.Map{
								"name":  pulumi.String("DD_ADDITIONAL_ENDPOINTS"),
								"value": pulumi.Sprintf(`{"%s": ["FAKEAPIKEY"]}`, fakeIntake.URL),
							},
						},
					},
					"clusterChecksRunner": pulumi.Map{
						"env": pulumi.MapArray{
							pulumi.Map{
								"name":  pulumi.String("DD_ADDITIONAL_ENDPOINTS"),
								"value": pulumi.Sprintf(`{"%s": ["FAKEAPIKEY"]}`, fakeIntake.URL),
							},
						},
					},
				}},
		})
		if err != nil {
			return nil, err
		}
	}

	err = mergo.Merge(&ddaArgs, defaultArgs)
	if err != nil {
		return nil, err
	}

	_, err = apiextensions.NewCustomResource(e.Ctx(), "datadog-agent", &apiextensions.CustomResourceArgs{
		ApiVersion: pulumi.String("datadoghq.com/v2alpha1"),
		Kind:       pulumi.String("DatadogAgent"),
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("datadog"),
			Namespace: pulumi.String("datadog"),
		},
		OtherFields: ddaArgs,
	}, opts...)
	if err != nil {
		return nil, err
	}

	return k8sComponent, nil
}
