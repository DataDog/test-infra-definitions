package k8s

import (
	// import embed
	_ "embed"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"

	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

//go:embed resources/prometheus-embed/go.mod_
var prometheus_go_mod string

//go:embed resources/prometheus-embed/go.sum_
var prometheus_go_sum string

//go:embed resources/prometheus-embed/main.go_
var prometheus_main_go string

func PrometheusWorkloadDefinition(e config.CommonEnvironment, kubeProvider *kubernetes.Provider, namespace string, opts ...pulumi.ResourceOption) (*appsv1.Deployment, error) {
	opts = append(opts, pulumi.Provider(kubeProvider))

	ns, err := corev1.NewNamespace(e.Ctx, namespace, &corev1.NamespaceArgs{
		Metadata: metav1.ObjectMetaArgs{
			Name: pulumi.String(namespace),
		},
	}, opts...)
	if err != nil {
		return nil, err
	}

	opts = append(opts, utils.PulumiDependsOn(ns))

	cm, err := corev1.NewConfigMap(e.Ctx, "prometheus", &corev1.ConfigMapArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("prometheus"),
			Namespace: pulumi.String(namespace),
			Labels: pulumi.StringMap{
				"app": pulumi.String("prometheus"),
			},
		},
		Data: pulumi.StringMap{
			"go.mod":  pulumi.String(prometheus_go_mod),
			"go.sum":  pulumi.String(prometheus_go_sum),
			"main.go": pulumi.String(prometheus_main_go),
		},
	}, opts...)
	if err != nil {
		return nil, err
	}

	opts = append(opts, utils.PulumiDependsOn(cm))

	deploy, err := appsv1.NewDeployment(e.Ctx, "prometheus", &appsv1.DeploymentArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("prometheus"),
			Namespace: pulumi.String(namespace),
			Labels: pulumi.StringMap{
				"app": pulumi.String("prometheus"),
			},
		},
		Spec: &appsv1.DeploymentSpecArgs{
			Replicas: pulumi.Int(1),
			Selector: &metav1.LabelSelectorArgs{
				MatchLabels: pulumi.StringMap{
					"app": pulumi.String("prometheus"),
				},
			},
			Template: &corev1.PodTemplateSpecArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Labels: pulumi.StringMap{
						"app": pulumi.String("prometheus"),
					},
					Annotations: pulumi.StringMap{
						"prometheus.io/scrape": pulumi.String("true"),
					},
				},
				Spec: &corev1.PodSpecArgs{
					Containers: corev1.ContainerArray{
						&corev1.ContainerArgs{
							Name:       pulumi.String("prometheus"),
							Image:      pulumi.String("golang:1.20"),
							WorkingDir: pulumi.String("/usr/src/prometheus"),
							Command: pulumi.StringArray{
								pulumi.String("go"),
								pulumi.String("run"),
								pulumi.String("."),
							},
							Resources: &corev1.ResourceRequirementsArgs{
								Limits: pulumi.StringMap{
									"cpu":    pulumi.String("1000m"),
									"memory": pulumi.String("256Mi"),
								},
								Requests: pulumi.StringMap{
									"cpu":    pulumi.String("10m"),
									"memory": pulumi.String("256Mi"),
								},
							},
							Ports: &corev1.ContainerPortArray{
								&corev1.ContainerPortArgs{
									Name:          pulumi.String("metrics"),
									ContainerPort: pulumi.Int(8080),
								},
							},
							LivenessProbe: &corev1.ProbeArgs{
								HttpGet: &corev1.HTTPGetActionArgs{
									Port: pulumi.Int(8080),
									Path: pulumi.StringPtr("/metrics"),
								},
							},
							ReadinessProbe: &corev1.ProbeArgs{
								HttpGet: &corev1.HTTPGetActionArgs{
									Port: pulumi.Int(8080),
									Path: pulumi.StringPtr("/metrics"),
								},
							},
							StartupProbe: &corev1.ProbeArgs{
								HttpGet: &corev1.HTTPGetActionArgs{
									Port: pulumi.Int(8080),
									Path: pulumi.StringPtr("/metrics"),
								},
								PeriodSeconds:    pulumi.IntPtr(10),
								FailureThreshold: pulumi.IntPtr(30),
							},
							VolumeMounts: &corev1.VolumeMountArray{
								&corev1.VolumeMountArgs{
									Name:      pulumi.String("prog"),
									MountPath: pulumi.String("/usr/src/prometheus"),
								},
							},
						},
					},
					Volumes: corev1.VolumeArray{
						&corev1.VolumeArgs{
							Name: pulumi.String("prog"),
							ConfigMap: &corev1.ConfigMapVolumeSourceArgs{
								Name: pulumi.String("prometheus"),
							},
						},
					},
				},
			},
		},
	}, append(opts, utils.PulumiDependsOn(cm))...)
	if err != nil {
		return nil, err
	}

	return deploy, nil
}
