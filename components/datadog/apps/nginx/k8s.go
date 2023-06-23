package nginx

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	ddv1alpha1 "github.com/DataDog/test-infra-definitions/components/kubernetes/crds/kubernetes/datadoghq/v1alpha1"

	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apps/v1"
	autoscalingv2 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/autoscaling/v2"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	policyv1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/policy/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type K8sComponent struct {
	pulumi.ResourceState
}

func K8sAppDefinition(e config.CommonEnvironment, kubeProvider *kubernetes.Provider, namespace string, dependsOnCrd pulumi.ResourceOption, opts ...pulumi.ResourceOption) (*K8sComponent, error) {
	opts = append(opts, pulumi.Provider(kubeProvider), pulumi.Parent(kubeProvider), pulumi.DeletedWith(kubeProvider))

	k8sComponent := &K8sComponent{}
	if err := e.Ctx.RegisterComponentResource("dd:apps", "nginx", k8sComponent, opts...); err != nil {
		return nil, err
	}

	opts = append(opts, pulumi.Parent(k8sComponent))

	ns, err := corev1.NewNamespace(e.Ctx, namespace, &corev1.NamespaceArgs{
		Metadata: metav1.ObjectMetaArgs{
			Name: pulumi.String(namespace),
		},
	}, opts...)
	if err != nil {
		return nil, err
	}

	opts = append(opts, utils.PulumiDependsOn(ns))

	if _, err := appsv1.NewDeployment(e.Ctx, "nginx", &appsv1.DeploymentArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("nginx"),
			Namespace: pulumi.String(namespace),
			Labels: pulumi.StringMap{
				"app": pulumi.String("nginx"),
			},
		},
		Spec: &appsv1.DeploymentSpecArgs{
			Selector: &metav1.LabelSelectorArgs{
				MatchLabels: pulumi.StringMap{
					"app": pulumi.String("nginx"),
				},
			},
			Template: &corev1.PodTemplateSpecArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Labels: pulumi.StringMap{
						"app": pulumi.String("nginx"),
					},
					Annotations: pulumi.StringMap{
						"ad.datadoghq.com/nginx.checks": pulumi.String(jsonMustMarshal(
							map[string]interface{}{
								"nginx": map[string]interface{}{
									"init_config": map[string]interface{}{},
									"instances": []map[string]interface{}{
										{
											"nginx_status_url": "http://%%host%%/nginx_status",
										},
									},
								},
							},
						)),
					},
				},
				Spec: &corev1.PodSpecArgs{
					Containers: corev1.ContainerArray{
						&corev1.ContainerArgs{
							Name:  pulumi.String("nginx"),
							Image: pulumi.String("ghcr.io/datadog/apps-nginx-server:main"),
							Resources: &corev1.ResourceRequirementsArgs{
								Limits: pulumi.StringMap{
									"cpu":    pulumi.String("100m"),
									"memory": pulumi.String("32Mi"),
								},
								Requests: pulumi.StringMap{
									"cpu":    pulumi.String("10m"),
									"memory": pulumi.String("32Mi"),
								},
							},
							Ports: &corev1.ContainerPortArray{
								&corev1.ContainerPortArgs{
									Name:          pulumi.String("http"),
									ContainerPort: pulumi.Int(80),
								},
							},
							LivenessProbe: &corev1.ProbeArgs{
								HttpGet: &corev1.HTTPGetActionArgs{
									Port: pulumi.Int(80),
								},
							},
							ReadinessProbe: &corev1.ProbeArgs{
								HttpGet: &corev1.HTTPGetActionArgs{
									Port: pulumi.Int(80),
								},
							},
							VolumeMounts: &corev1.VolumeMountArray{
								&corev1.VolumeMountArgs{
									Name:      pulumi.String("cache"),
									MountPath: pulumi.String("/var/cache/nginx"),
								},
								&corev1.VolumeMountArgs{
									Name:      pulumi.String("var-run"),
									MountPath: pulumi.String("/var/run"),
								},
							},
						},
					},
					Volumes: corev1.VolumeArray{
						&corev1.VolumeArgs{
							Name:     pulumi.String("cache"),
							EmptyDir: &corev1.EmptyDirVolumeSourceArgs{},
						},
						&corev1.VolumeArgs{
							Name:     pulumi.String("var-run"),
							EmptyDir: &corev1.EmptyDirVolumeSourceArgs{},
						},
					},
				},
			},
		},
	}, opts...); err != nil {
		return nil, err
	}

	if _, err := policyv1.NewPodDisruptionBudget(e.Ctx, "nginx", &policyv1.PodDisruptionBudgetArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("nginx"),
			Namespace: pulumi.String(namespace),
			Labels: pulumi.StringMap{
				"app": pulumi.String("nginx"),
			},
		},
		Spec: &policyv1.PodDisruptionBudgetSpecArgs{
			MaxUnavailable: pulumi.Int(1),
			Selector: &metav1.LabelSelectorArgs{
				MatchLabels: pulumi.StringMap{
					"app": pulumi.String("nginx"),
				},
			},
		},
	}, opts...); err != nil {
		return nil, err
	}

	if dependsOnCrd != nil {
		ddm, err := ddv1alpha1.NewDatadogMetric(e.Ctx, "nginx", &ddv1alpha1.DatadogMetricArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("nginx"),
				Namespace: pulumi.String(namespace),
				Labels: pulumi.StringMap{
					"app": pulumi.String("nginx"),
				},
			},
			Spec: &ddv1alpha1.DatadogMetricSpecArgs{
				Query: pulumi.String(fmt.Sprintf("avg:nginx.net.request_per_s{kube_cluster_name:%%%%tag_kube_cluster_name%%%%,kube_namespace:%s,kube_deployment:nginx}.rollup(60)", namespace)),
			},
		}, append(opts, dependsOnCrd)...)
		if err != nil {
			return nil, err
		}

		if _, err := autoscalingv2.NewHorizontalPodAutoscaler(e.Ctx, "nginx", &autoscalingv2.HorizontalPodAutoscalerArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("nginx"),
				Namespace: pulumi.String(namespace),
				Labels: pulumi.StringMap{
					"app": pulumi.String("nginx"),
				},
			},
			Spec: &autoscalingv2.HorizontalPodAutoscalerSpecArgs{
				MinReplicas: pulumi.Int(1),
				MaxReplicas: pulumi.Int(5),
				ScaleTargetRef: &autoscalingv2.CrossVersionObjectReferenceArgs{
					ApiVersion: pulumi.String("apps/v1"),
					Kind:       pulumi.String("Deployment"),
					Name:       pulumi.String("nginx"),
				},
				Metrics: &autoscalingv2.MetricSpecArray{
					&autoscalingv2.MetricSpecArgs{
						Type: pulumi.String("External"),
						External: &autoscalingv2.ExternalMetricSourceArgs{
							Metric: &autoscalingv2.MetricIdentifierArgs{
								Name: pulumi.String("datadogmetric@" + namespace + ":nginx"),
							},
							Target: &autoscalingv2.MetricTargetArgs{
								Type:         pulumi.String("AverageValue"),
								AverageValue: pulumi.String("10"),
							},
						},
					},
				},
			},
		}, append(opts, utils.PulumiDependsOn(ddm))...); err != nil {
			return nil, err
		}
	}

	if _, err := corev1.NewService(e.Ctx, "nginx", &corev1.ServiceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("nginx"),
			Namespace: pulumi.String(namespace),
			Labels: pulumi.StringMap{
				"app": pulumi.String("nginx"),
			},
			Annotations: pulumi.StringMap{
				"ad.datadoghq.com/service.checks": pulumi.String(jsonMustMarshal(
					map[string]interface{}{
						"http_check": map[string]interface{}{
							"init_config": map[string]interface{}{},
							"instances": []map[string]interface{}{
								{
									"name":    "My Nginx",
									"url":     "http://%%host%%",
									"timeout": 1,
								},
							},
						},
					},
				)),
			},
		},
		Spec: &corev1.ServiceSpecArgs{
			Selector: pulumi.StringMap{
				"app": pulumi.String("nginx"),
			},
			Ports: &corev1.ServicePortArray{
				&corev1.ServicePortArgs{
					Name:       pulumi.String("http"),
					Port:       pulumi.Int(80),
					TargetPort: pulumi.String("http"),
				},
			},
		},
	}, opts...); err != nil {
		return nil, err
	}

	if _, err := appsv1.NewDeployment(e.Ctx, "nginx-query", &appsv1.DeploymentArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("nginx-query"),
			Namespace: pulumi.String(namespace),
			Labels: pulumi.StringMap{
				"app": pulumi.String("nginx-query"),
			},
		},
		Spec: &appsv1.DeploymentSpecArgs{
			Replicas: pulumi.Int(1),
			Selector: &metav1.LabelSelectorArgs{
				MatchLabels: pulumi.StringMap{
					"app": pulumi.String("nginx-query"),
				},
			},
			Template: &corev1.PodTemplateSpecArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Labels: pulumi.StringMap{
						"app": pulumi.String("nginx-query"),
					},
				},
				Spec: &corev1.PodSpecArgs{
					Containers: corev1.ContainerArray{
						&corev1.ContainerArgs{
							Name:  pulumi.String("query"),
							Image: pulumi.String("ghcr.io/datadog/apps-http-client:main"),
							Resources: &corev1.ResourceRequirementsArgs{
								Limits: pulumi.StringMap{
									"cpu":    pulumi.String("100m"),
									"memory": pulumi.String("32Mi"),
								},
								Requests: pulumi.StringMap{
									"cpu":    pulumi.String("10m"),
									"memory": pulumi.String("32Mi"),
								},
							},
						},
					},
				},
			},
		},
	}, opts...); err != nil {
		return nil, err
	}

	return k8sComponent, nil
}
