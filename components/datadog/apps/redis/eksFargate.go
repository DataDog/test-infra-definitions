package redis

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
	ddv1alpha1 "github.com/DataDog/test-infra-definitions/components/kubernetes/crds/kubernetes/datadoghq/v1alpha1"
	"github.com/Masterminds/semver"

	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	autoscalingv2 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/autoscaling/v2"
	autoscalingv2beta2 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/autoscaling/v2beta2"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	policyv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/policy/v1"
	policyv1beta1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/policy/v1beta1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type EKSFargateComponent struct {
	pulumi.ResourceState
}

func EKSFargateAppDefinition(e config.CommonEnvironment, namespace string, dependsOnCrd pulumi.ResourceOption, fakeIntakeParam *fakeintake.Fakeintake, opts ...pulumi.ResourceOption) (*EKSFargateComponent, error) {
	eksFargateComponent := &EKSFargateComponent{}
	if err := e.Ctx.RegisterComponentResource("dd:apps", "redis-fargate", eksFargateComponent, opts...); err != nil {
		return nil, err
	}

	kubeVersion, err := semver.NewVersion(e.KubernetesVersion())
	if err != nil {
		return nil, err
	}

	opts = append(opts, pulumi.Parent(eksFargateComponent))

	// start redis pod
	if _, err := appsv1.NewDeployment(e.Ctx, "redis-fargate", &appsv1.DeploymentArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("redis"),
			Namespace: pulumi.String(namespace),
			Labels: pulumi.StringMap{
				"app":                             pulumi.String("redis"),
				"agent.datadoghq.com/sidecar":     pulumi.String("fargate"),
				"admission.datadoghq.com/enabled": pulumi.String("false"),
				"injectSidecarPodLabel":           pulumi.String("true"),
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
						"app":                             pulumi.String("redis"),
						"agent.datadoghq.com/sidecar":     pulumi.String("fargate"),
						"admission.datadoghq.com/enabled": pulumi.String("false"),
						"injectSidecarPodLabel":           pulumi.String("true"),
					},
				},
				Spec: &corev1.PodSpecArgs{
					ServiceAccountName: pulumi.String("datadog-agent"),
					Containers: corev1.ContainerArray{
						&corev1.ContainerArgs{
							Name:  pulumi.String("redis"),
							Image: pulumi.String("public.ecr.aws/docker/library/redis:latest"),
							Args: pulumi.StringArray{
								pulumi.String("--loglevel"),
								pulumi.String("verbose"),
							},
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
									Protocol:      pulumi.String("TCP"),
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

	// In versions older than 1.21.0, we should use policyv1beta1
	kubeThresholdVersion, _ := semver.NewVersion("1.21.0")

	if kubeVersion.Compare(kubeThresholdVersion) < 0 {
		if _, err := policyv1beta1.NewPodDisruptionBudget(e.Ctx, "redis-fargate", &policyv1beta1.PodDisruptionBudgetArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("redis"),
				Namespace: pulumi.String(namespace),
				Labels: pulumi.StringMap{
					"app": pulumi.String("redis"),
				},
			},
			Spec: &policyv1beta1.PodDisruptionBudgetSpecArgs{
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
	} else {
		if _, err := policyv1.NewPodDisruptionBudget(e.Ctx, "redis-fargate", &policyv1.PodDisruptionBudgetArgs{
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
	}

	if dependsOnCrd != nil {
		ddm, err := ddv1alpha1.NewDatadogMetric(e.Ctx, "redis-fargate", &ddv1alpha1.DatadogMetricArgs{
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
		}, append(opts, dependsOnCrd)...)
		if err != nil {
			return nil, err
		}

		// In versions older than 1.23.0, we should use autoscalingv2beta2
		kubeThresholdVersion, _ = semver.NewVersion("1.23.0")

		if kubeVersion.Compare(kubeThresholdVersion) < 0 {
			if _, err := autoscalingv2beta2.NewHorizontalPodAutoscaler(e.Ctx, "redis-fargate", &autoscalingv2beta2.HorizontalPodAutoscalerArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name:      pulumi.String("redis"),
					Namespace: pulumi.String(namespace),
					Labels: pulumi.StringMap{
						"app": pulumi.String("redis"),
					},
				},
				Spec: &autoscalingv2beta2.HorizontalPodAutoscalerSpecArgs{
					MinReplicas: pulumi.Int(1),
					MaxReplicas: pulumi.Int(5),
					ScaleTargetRef: &autoscalingv2beta2.CrossVersionObjectReferenceArgs{
						ApiVersion: pulumi.String("apps/v1"),
						Kind:       pulumi.String("Deployment"),
						Name:       pulumi.String("redis"),
					},
					Metrics: &autoscalingv2beta2.MetricSpecArray{
						&autoscalingv2beta2.MetricSpecArgs{
							Type: pulumi.String("External"),
							External: &autoscalingv2beta2.ExternalMetricSourceArgs{
								Metric: &autoscalingv2beta2.MetricIdentifierArgs{
									Name: pulumi.String("datadogmetric@" + namespace + ":redis"),
								},
								Target: &autoscalingv2beta2.MetricTargetArgs{
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
		} else {
			if _, err := autoscalingv2.NewHorizontalPodAutoscaler(e.Ctx, "redis-fargate", &autoscalingv2.HorizontalPodAutoscalerArgs{
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
		}

	}

	if _, err := corev1.NewService(e.Ctx, "redis-fargate", &corev1.ServiceArgs{
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
					Protocol:   pulumi.String("TCP"),
				},
			},
		},
	}, opts...); err != nil {
		return nil, err
	}

	if _, err := appsv1.NewDeployment(e.Ctx, "redis-query-fargate", &appsv1.DeploymentArgs{
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
							Args: pulumi.StringArray{
								pulumi.String("-min-tps"),
								pulumi.String("1"),
								pulumi.String("-max-tps"),
								pulumi.String("60"),
								pulumi.String("-period"),
								pulumi.String("20m"),
							},
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

	return eksFargateComponent, nil
}
