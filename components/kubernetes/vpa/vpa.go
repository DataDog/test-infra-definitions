package vpa

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	apiextensions "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apiextensions/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/DataDog/test-infra-definitions/common/config"
)

func DeployCRD(e config.Env, kubeProvider *kubernetes.Provider, opts ...pulumi.ResourceOption) (*apiextensions.CustomResourceDefinition, error) {
	opts = append(opts, pulumi.Providers(kubeProvider), pulumi.DeletedWith(kubeProvider))

	// This is the definition from https://github.com/kubernetes/autoscaler/blob/4d092e5f0afd519b082972c4656b3d52ae512b64/vertical-pod-autoscaler/deploy/vpa-v1-crd.yaml
	return apiextensions.NewCustomResourceDefinition(e.Ctx(), "vertical-pod-autoscaler", &apiextensions.CustomResourceDefinitionArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name: pulumi.String("verticalpodautoscalers.autoscaling.k8s.io"),
			Annotations: pulumi.StringMap{
				"api-approved.kubernetes.io": pulumi.String("https://github.com/kubernetes/kubernetes/pull/63797"),
			},
		},
		Spec: &apiextensions.CustomResourceDefinitionSpecArgs{
			Group: pulumi.String("autoscaling.k8s.io"),
			Scope: pulumi.String("Namespaced"),
			Names: &apiextensions.CustomResourceDefinitionNamesArgs{
				Kind:       pulumi.String("VerticalPodAutoscaler"),
				ListKind:   pulumi.String("VerticalPodAutoscalerList"),
				Singular:   pulumi.String("verticalpodautoscaler"),
				Plural:     pulumi.String("verticalpodautoscalers"),
				ShortNames: pulumi.StringArray{pulumi.String("vpa")},
			},
			Versions: &apiextensions.CustomResourceDefinitionVersionArray{
				&apiextensions.CustomResourceDefinitionVersionArgs{
					Name:    pulumi.String("v1"),
					Served:  pulumi.Bool(true),
					Storage: pulumi.Bool(true),
					Schema: &apiextensions.CustomResourceValidationArgs{
						OpenAPIV3Schema: &apiextensions.JSONSchemaPropsArgs{
							Type: pulumi.String("object"),
							Properties: apiextensions.JSONSchemaPropsMap{
								"spec": apiextensions.JSONSchemaPropsArgs{
									Type: pulumi.String("object"),
									Properties: apiextensions.JSONSchemaPropsMap{
										"targetRef": apiextensions.JSONSchemaPropsArgs{
											Type: pulumi.String("object"),
										},
										"updatePolicy": apiextensions.JSONSchemaPropsArgs{
											Type: pulumi.String("object"),
											Properties: apiextensions.JSONSchemaPropsMap{
												"minReplicas": apiextensions.JSONSchemaPropsArgs{
													Type: pulumi.String("integer"),
												},
												"updateMode": apiextensions.JSONSchemaPropsArgs{
													Type: pulumi.String("string"),
												},
											},
										},
										"resourcePolicy": apiextensions.JSONSchemaPropsArgs{
											Type: pulumi.String("object"),
											Properties: apiextensions.JSONSchemaPropsMap{
												"containerPolicies": apiextensions.JSONSchemaPropsArgs{
													Type: pulumi.String("array"),
													Items: &apiextensions.JSONSchemaPropsArgs{
														Type: pulumi.String("object"),
														Properties: apiextensions.JSONSchemaPropsMap{
															"containerName": apiextensions.JSONSchemaPropsArgs{
																Type: pulumi.String("string"),
															},
															"controlledValue": apiextensions.JSONSchemaPropsArgs{
																Type: pulumi.String("string"),
																Enum: pulumi.Array{
																	pulumi.String("RequestsAndLimits"),
																	pulumi.String("RequestsOnly"),
																},
															},
															"mode": apiextensions.JSONSchemaPropsArgs{
																Type: pulumi.String("string"),
																Enum: pulumi.Array{
																	pulumi.String("Auto"),
																	pulumi.String("Off"),
																},
															},
															"minAllowed": apiextensions.JSONSchemaPropsArgs{
																Type: pulumi.String("object"),
															},
															"maxAllowed": apiextensions.JSONSchemaPropsArgs{
																Type: pulumi.String("object"),
															},
															"controlledResources": apiextensions.JSONSchemaPropsArgs{
																Type: pulumi.String("array"),
																Items: &apiextensions.JSONSchemaPropsArgs{
																	Type: pulumi.String("string"),
																	Enum: pulumi.Array{
																		pulumi.String("cpu"),
																		pulumi.String("memory"),
																	},
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}, opts...)
}
