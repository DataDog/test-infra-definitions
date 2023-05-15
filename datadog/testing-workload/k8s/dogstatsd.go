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

//go:embed resources/dogstatsd-embed/go.mod_
var dogstatsd_go_mod string

//go:embed resources/dogstatsd-embed/go.sum_
var dogstatsd_go_sum string

//go:embed resources/dogstatsd-embed/main.go_
var dogstatsd_main_go string

func DogstatsdWorkloadDefinition(e config.CommonEnvironment, kubeProvider *kubernetes.Provider, namespace string, opts ...pulumi.ResourceOption) (*appsv1.Deployment, *appsv1.Deployment, error) {
	opts = append(opts, pulumi.Provider(kubeProvider))

	ns, err := corev1.NewNamespace(e.Ctx, namespace, &corev1.NamespaceArgs{
		Metadata: metav1.ObjectMetaArgs{
			Name: pulumi.String(namespace),
		},
	}, opts...)
	if err != nil {
		return nil, nil, err
	}

	opts = append(opts, utils.PulumiDependsOn(ns))

	cm, err := corev1.NewConfigMap(e.Ctx, "dogstatsd", &corev1.ConfigMapArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("dogstatsd"),
			Namespace: pulumi.String(namespace),
			Labels: pulumi.StringMap{
				"app": pulumi.String("dogstatsd"),
			},
		},
		Data: pulumi.StringMap{
			"go.mod":  pulumi.String(dogstatsd_go_mod),
			"go.sum":  pulumi.String(dogstatsd_go_sum),
			"main.go": pulumi.String(dogstatsd_main_go),
		},
	}, opts...)
	if err != nil {
		return nil, nil, err
	}

	opts = append(opts, utils.PulumiDependsOn(cm))

	uds, err := appsv1.NewDeployment(e.Ctx, "dogstatsd-uds", &appsv1.DeploymentArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("dogstatsd-uds"),
			Namespace: pulumi.String(namespace),
			Labels: pulumi.StringMap{
				"app": pulumi.String("dogstatsd-uds"),
			},
		},
		Spec: &appsv1.DeploymentSpecArgs{
			Replicas: pulumi.Int(1),
			Selector: &metav1.LabelSelectorArgs{
				MatchLabels: pulumi.StringMap{
					"app": pulumi.String("dogstatsd-uds"),
				},
			},
			Template: &corev1.PodTemplateSpecArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Labels: pulumi.StringMap{
						"app": pulumi.String("dogstatsd-uds"),
					},
				},
				Spec: &corev1.PodSpecArgs{
					Containers: corev1.ContainerArray{
						&corev1.ContainerArgs{
							Name:       pulumi.String("dogstatsd"),
							Image:      pulumi.String("golang:1.20"),
							WorkingDir: pulumi.String("/usr/src/dogstatsd"),
							Command: pulumi.StringArray{
								pulumi.String("go"),
								pulumi.String("run"),
								pulumi.String("."),
							},
							Env: &corev1.EnvVarArray{
								&corev1.EnvVarArgs{
									Name:  pulumi.String("STATSD_URL"),
									Value: pulumi.StringPtr("unix:///var/run/datadog/dsd.socket"),
								},
								&corev1.EnvVarArgs{
									Name:  pulumi.String("SLEEP"),
									Value: pulumi.StringPtr("1s"),
								},
							},
							Resources: &corev1.ResourceRequirementsArgs{
								Limits: pulumi.StringMap{
									"cpu":    pulumi.String("10m"),
									"memory": pulumi.String("256Mi"),
								},
								Requests: pulumi.StringMap{
									"cpu":    pulumi.String("2m"),
									"memory": pulumi.String("256Mi"),
								},
							},
							VolumeMounts: &corev1.VolumeMountArray{
								&corev1.VolumeMountArgs{
									Name:      pulumi.String("prog"),
									MountPath: pulumi.String("/usr/src/dogstatsd"),
								},
								&corev1.VolumeMountArgs{
									Name:      pulumi.String("var-run-datadog"),
									MountPath: pulumi.String("/var/run/datadog"),
								},
							},
						},
					},
					Volumes: corev1.VolumeArray{
						&corev1.VolumeArgs{
							Name: pulumi.String("prog"),
							ConfigMap: &corev1.ConfigMapVolumeSourceArgs{
								Name: pulumi.String("dogstatsd"),
							},
						},
						&corev1.VolumeArgs{
							Name: pulumi.String("var-run-datadog"),
							HostPath: &corev1.HostPathVolumeSourceArgs{
								Path: pulumi.String("/var/run/datadog"),
								Type: pulumi.StringPtr("Directory"),
							},
						},
					},
				},
			},
		},
	}, append(opts, utils.PulumiDependsOn(cm))...)
	if err != nil {
		return nil, nil, err
	}

	udp, err := appsv1.NewDeployment(e.Ctx, "dogstatsd-udp", &appsv1.DeploymentArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("dogstatsd-udp"),
			Namespace: pulumi.String(namespace),
			Labels: pulumi.StringMap{
				"app": pulumi.String("dogstatsd-udp"),
			},
		},
		Spec: &appsv1.DeploymentSpecArgs{
			Replicas: pulumi.Int(1),
			Selector: &metav1.LabelSelectorArgs{
				MatchLabels: pulumi.StringMap{
					"app": pulumi.String("dogstatsd-udp"),
				},
			},
			Template: &corev1.PodTemplateSpecArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Labels: pulumi.StringMap{
						"app": pulumi.String("dogstatsd-udp"),
					},
				},
				Spec: &corev1.PodSpecArgs{
					Containers: corev1.ContainerArray{
						&corev1.ContainerArgs{
							Name:       pulumi.String("dogstatsd"),
							Image:      pulumi.String("golang:1.20"),
							WorkingDir: pulumi.String("/usr/src/dogstatsd"),
							Command: pulumi.StringArray{
								pulumi.String("go"),
								pulumi.String("run"),
								pulumi.String("."),
							},
							Env: &corev1.EnvVarArray{
								&corev1.EnvVarArgs{
									Name: pulumi.String("HOST_IP"),
									ValueFrom: &corev1.EnvVarSourceArgs{
										FieldRef: &corev1.ObjectFieldSelectorArgs{
											FieldPath: pulumi.String("status.hostIP"),
										},
									},
								},
								&corev1.EnvVarArgs{
									Name:  pulumi.String("STATSD_URL"),
									Value: pulumi.StringPtr("$(HOST_IP):8125"),
								},
								&corev1.EnvVarArgs{
									Name: pulumi.String("DD_ENTITY_ID"),
									ValueFrom: &corev1.EnvVarSourceArgs{
										FieldRef: &corev1.ObjectFieldSelectorArgs{
											FieldPath: pulumi.String("metadata.uid"),
										},
									},
								},
								&corev1.EnvVarArgs{
									Name:  pulumi.String("SLEEP"),
									Value: pulumi.StringPtr("1s"),
								},
							},
							Resources: &corev1.ResourceRequirementsArgs{
								Limits: pulumi.StringMap{
									"cpu":    pulumi.String("10m"),
									"memory": pulumi.String("256Mi"),
								},
								Requests: pulumi.StringMap{
									"cpu":    pulumi.String("2m"),
									"memory": pulumi.String("256Mi"),
								},
							},
							VolumeMounts: &corev1.VolumeMountArray{
								&corev1.VolumeMountArgs{
									Name:      pulumi.String("prog"),
									MountPath: pulumi.String("/usr/src/dogstatsd"),
								},
							},
						},
					},
					Volumes: corev1.VolumeArray{
						&corev1.VolumeArgs{
							Name: pulumi.String("prog"),
							ConfigMap: &corev1.ConfigMapVolumeSourceArgs{
								Name: pulumi.String("dogstatsd"),
							},
						},
					},
				},
			},
		},
	}, append(opts, utils.PulumiDependsOn(cm))...)
	if err != nil {
		return nil, nil, err
	}

	return uds, udp, nil
}
