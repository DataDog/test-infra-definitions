package redis

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

func K8sAppDefinition(e config.CommonEnvironment, kubeProvider *kubernetes.Provider, namespace string, opts ...pulumi.ResourceOption) (*K8sComponent, error) {
	opts = append(opts, pulumi.Provider(kubeProvider), pulumi.Parent(kubeProvider), pulumi.DeletedWith(kubeProvider))

	k8sComponent := &K8sComponent{}
	if err := e.Ctx.RegisterComponentResource("dd:apps", "redis", k8sComponent, opts...); err != nil {
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

	if _, err := appsv1.NewDeployment(e.Ctx, "redis", &appsv1.DeploymentArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("redis"),
			Namespace: pulumi.String(namespace),
			Labels: pulumi.StringMap{
				"app": pulumi.String("redis"),
			},
		},
		Spec: &appsv1.DeploymentSpecArgs{
			Selector: &metav1.LabelSelectorArgs{
				MatchLabels: pulumi.StringMap{
					"app": pulumi.String("redis"),
				},
			},
			Template: &corev1.PodTemplateSpecArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Labels: pulumi.StringMap{
						"app": pulumi.String("redis"),
					},
				},
				Spec: &corev1.PodSpecArgs{
					Containers: corev1.ContainerArray{
						&corev1.ContainerArgs{
							Name:  pulumi.String("redis"),
							Image: pulumi.String("redis:latest"),
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
									Name:          pulumi.String("redis"),
									ContainerPort: pulumi.Int(6379),
								},
							},
							LivenessProbe: &corev1.ProbeArgs{
								TcpSocket: &corev1.TCPSocketActionArgs{
									Port: pulumi.Int(6379),
								},
							},
							ReadinessProbe: &corev1.ProbeArgs{
								TcpSocket: &corev1.TCPSocketActionArgs{
									Port: pulumi.Int(6379),
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

	if _, err := policyv1.NewPodDisruptionBudget(e.Ctx, "redis", &policyv1.PodDisruptionBudgetArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("redis"),
			Namespace: pulumi.String(namespace),
			Labels: pulumi.StringMap{
				"app": pulumi.String("redis"),
			},
		},
		Spec: &policyv1.PodDisruptionBudgetSpecArgs{
			MaxUnavailable: pulumi.Int(1),
			Selector: &metav1.LabelSelectorArgs{
				MatchLabels: pulumi.StringMap{
					"app": pulumi.String("redis"),
				},
			},
		},
	}, opts...); err != nil {
		return nil, err
	}

	ddm, err := ddv1alpha1.NewDatadogMetric(e.Ctx, "redis", &ddv1alpha1.DatadogMetricArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("redis"),
			Namespace: pulumi.String(namespace),
			Labels: pulumi.StringMap{
				"app": pulumi.String("redis"),
			},
		},
		Spec: &ddv1alpha1.DatadogMetricSpecArgs{
			Query: pulumi.String(fmt.Sprintf("avg:redis.net.instantaneous_ops_per_sec{kube_cluster_name:%%%%tag_kube_cluster_name%%%%,kube_namespace:%s,kube_deployment:redis}.rollup(60)", namespace)),
		},
	}, opts...)
	if err != nil {
		return nil, err
	}

	if _, err := autoscalingv2.NewHorizontalPodAutoscaler(e.Ctx, "redis", &autoscalingv2.HorizontalPodAutoscalerArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("redis"),
			Namespace: pulumi.String(namespace),
			Labels: pulumi.StringMap{
				"app": pulumi.String("redis"),
			},
		},
		Spec: &autoscalingv2.HorizontalPodAutoscalerSpecArgs{
			MinReplicas: pulumi.Int(1),
			MaxReplicas: pulumi.Int(5),
			ScaleTargetRef: &autoscalingv2.CrossVersionObjectReferenceArgs{
				ApiVersion: pulumi.String("apps/v1"),
				Kind:       pulumi.String("Deployment"),
				Name:       pulumi.String("redis"),
			},
			Metrics: &autoscalingv2.MetricSpecArray{
				&autoscalingv2.MetricSpecArgs{
					Type: pulumi.String("External"),
					External: &autoscalingv2.ExternalMetricSourceArgs{
						Metric: &autoscalingv2.MetricIdentifierArgs{
							Name: pulumi.String("datadogmetric@" + namespace + ":redis"),
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

	if _, err := corev1.NewService(e.Ctx, "redis", &corev1.ServiceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("redis"),
			Namespace: pulumi.String(namespace),
			Labels: pulumi.StringMap{
				"app": pulumi.String("redis"),
			},
		},
		Spec: &corev1.ServiceSpecArgs{
			Selector: pulumi.StringMap{
				"app": pulumi.String("redis"),
			},
			Ports: &corev1.ServicePortArray{
				&corev1.ServicePortArgs{
					Name:       pulumi.String("redis"),
					Port:       pulumi.Int(6379),
					TargetPort: pulumi.String("redis"),
				},
			},
		},
	}, opts...); err != nil {
		return nil, err
	}

	if _, err := appsv1.NewDeployment(e.Ctx, "redis-query", &appsv1.DeploymentArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("redis-query"),
			Namespace: pulumi.String(namespace),
			Labels: pulumi.StringMap{
				"app": pulumi.String("redis-query"),
			},
		},
		Spec: &appsv1.DeploymentSpecArgs{
			Replicas: pulumi.Int(1),
			Selector: &metav1.LabelSelectorArgs{
				MatchLabels: pulumi.StringMap{
					"app": pulumi.String("redis-query"),
				},
			},
			Template: &corev1.PodTemplateSpecArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Labels: pulumi.StringMap{
						"app": pulumi.String("redis-query"),
					},
				},
				Spec: &corev1.PodSpecArgs{
					Containers: corev1.ContainerArray{
						&corev1.ContainerArgs{
							Name:  pulumi.String("query"),
							Image: pulumi.String("ghcr.io/datadog/apps-redis-client:main"),
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
