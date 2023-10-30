package dogstatsdstandalone

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	ddfakeintake "github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	v1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/rbac/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// HostPort defines the port used by the dogstatsd standalone deployment. The
// standard port for dogstatsd is 8125, but it's already used by the agent.
const HostPort = 8128

// Socket defines the socket exposed by the dogstatsd standalone deployment.
// It's not the default to avoid conflict with the agent.
const Socket = "/var/run/datadog/dsd-standalone.socket"

type K8sComponent struct {
	pulumi.ResourceState
}

func K8sAppDefinition(e config.CommonEnvironment, kubeProvider *kubernetes.Provider, namespace string, fakeIntake *ddfakeintake.ConnectionExporter, opts ...pulumi.ResourceOption) (*K8sComponent, error) {
	opts = append(opts, pulumi.Provider(kubeProvider), pulumi.Parent(kubeProvider), pulumi.DeletedWith(kubeProvider))

	k8sComponent := &K8sComponent{}
	if err := e.Ctx.RegisterComponentResource("dd:dogstatsd-standalone", "dogstatsd", k8sComponent, opts...); err != nil {
		return nil, err
	}

	opts = append(opts, pulumi.Parent(k8sComponent))

	ns, err := corev1.NewNamespace(
		e.Ctx,
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

	envVars := corev1.EnvVarArray{
		&corev1.EnvVarArgs{
			Name:  pulumi.String("DD_API_KEY"),
			Value: e.AgentAPIKey(),
		},
		&corev1.EnvVarArgs{
			Name: pulumi.String("DD_KUBERNETES_KUBELET_HOST"),
			ValueFrom: corev1.EnvVarSourceArgs{
				FieldRef: corev1.ObjectFieldSelectorArgs{
					FieldPath: pulumi.String("status.hostIP"),
				},
			},
		},
		&corev1.EnvVarArgs{
			Name:  pulumi.String("DD_DOGSTATSD_NON_LOCAL_TRAFFIC"),
			Value: pulumi.StringPtr("true"),
		},
		&corev1.EnvVarArgs{
			Name:  pulumi.String("DD_DOGSTATSD_ORIGIN_DETECTION"),
			Value: pulumi.StringPtr("true"),
		},
		&corev1.EnvVarArgs{
			Name:  pulumi.String("DD_DOGSTATSD_SOCKET"),
			Value: pulumi.StringPtr(Socket),
		},
		&corev1.EnvVarArgs{
			Name:  pulumi.String("DD_DOGSTATSD_TAG_CARDINALITY"),
			Value: pulumi.StringPtr("orchestrator"),
		},
		&corev1.EnvVarArgs{
			Name:  pulumi.String("DD_KUBELET_TLS_VERIFY"),
			Value: pulumi.StringPtr("false"),
		},
	}

	if fakeIntake != nil {
		envVars = append(
			envVars,
			&corev1.EnvVarArgs{
				Name:  pulumi.String("DD_ADDITIONAL_ENDPOINTS"),
				Value: pulumi.Sprintf(`{"http://%s": ["FAKEAPIKEY"]}`, fakeIntake.Host),
			},
		)
	}

	daemonSetArgs := appsv1.DaemonSetArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("dogstatsd-standalone"),
			Namespace: pulumi.String(namespace),
		},
		Spec: &appsv1.DaemonSetSpecArgs{
			Selector: &metav1.LabelSelectorArgs{
				MatchLabels: pulumi.StringMap{
					"app": pulumi.String("dogstatsd-standalone"),
				},
			},
			Template: &corev1.PodTemplateSpecArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Labels: pulumi.StringMap{
						"app": pulumi.String("dogstatsd-standalone"),
					},
				},
				Spec: &corev1.PodSpecArgs{
					HostPID:            pulumi.BoolPtr(true),
					ServiceAccountName: pulumi.String("dogstatsd-standalone"),
					Containers: corev1.ContainerArray{
						&corev1.ContainerArgs{
							Name:  pulumi.String("dogstatsd-standalone"),
							Image: pulumi.String(e.DogstatsdFullImagePath()),
							Ports: corev1.ContainerPortArray{
								&corev1.ContainerPortArgs{
									ContainerPort: pulumi.Int(8125),
									HostPort:      pulumi.Int(HostPort),
									Name:          pulumi.StringPtr("dogstatsdport"),
									Protocol:      pulumi.StringPtr("UDP"),
								},
							},
							Env: &envVars,
							Resources: &corev1.ResourceRequirementsArgs{
								Limits: pulumi.StringMap{
									"cpu":    pulumi.String("100m"),
									"memory": pulumi.String("512Mi"),
								},
								Requests: pulumi.StringMap{
									"cpu":    pulumi.String("100m"),
									"memory": pulumi.String("512Mi"),
								},
							},
							VolumeMounts: &corev1.VolumeMountArray{
								&corev1.VolumeMountArgs{
									Name:      pulumi.String("hostvar"),
									MountPath: pulumi.String("/host/var"),
									ReadOnly:  pulumi.BoolPtr(true),
								},
								&corev1.VolumeMountArgs{
									Name:      pulumi.String("procdir"),
									MountPath: pulumi.String("/host/proc"),
									ReadOnly:  pulumi.BoolPtr(true),
								},
								&corev1.VolumeMountArgs{
									Name:      pulumi.String("cgroups"),
									MountPath: pulumi.String("/host/sys/fs/cgroup"),
									ReadOnly:  pulumi.BoolPtr(true),
								},
								&corev1.VolumeMountArgs{
									Name:      pulumi.String("dsdsocket"),
									MountPath: pulumi.String("/var/run/datadog"),
								},
							},
						},
					},
					Volumes: corev1.VolumeArray{
						&corev1.VolumeArgs{
							Name: pulumi.String("hostvar"),
							HostPath: &corev1.HostPathVolumeSourceArgs{
								Path: pulumi.String("/var"),
							},
						},
						&corev1.VolumeArgs{
							Name: pulumi.String("procdir"),
							HostPath: &corev1.HostPathVolumeSourceArgs{
								Path: pulumi.String("/proc"),
							},
						},
						&corev1.VolumeArgs{
							Name: pulumi.String("cgroups"),
							HostPath: &corev1.HostPathVolumeSourceArgs{
								Path: pulumi.String("/sys/fs/cgroup"),
							},
						},
						&corev1.VolumeArgs{
							Name: pulumi.String("dsdsocket"),
							HostPath: &corev1.HostPathVolumeSourceArgs{
								Path: pulumi.String("/var/run/datadog/"),
							},
						},
					},
				},
			},
		},
	}

	serviceAccountArgs := corev1.ServiceAccountArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("dogstatsd-standalone"),
			Namespace: pulumi.String(namespace),
		},
	}

	clusterRoleArgs := v1.ClusterRoleArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name: pulumi.String("dogstatsd-standalone"),
		},
		Rules: v1.PolicyRuleArray{
			v1.PolicyRuleArgs{
				NonResourceURLs: pulumi.StringArray{
					pulumi.String("/metrics"),
				},
				Verbs: pulumi.StringArray{
					pulumi.String("get"),
				},
			},
			v1.PolicyRuleArgs{ // Kubelet connectivity
				ApiGroups: pulumi.StringArray{
					pulumi.String(""),
				},
				Resources: pulumi.StringArray{
					pulumi.String("nodes/metrics"),
					pulumi.String("nodes/spec"),
					pulumi.String("nodes/proxy"),
					pulumi.String("nodes/stats"),
				},
				Verbs: pulumi.StringArray{
					pulumi.String("get"),
				},
			},
		},
	}

	clusterRoleBindingArgs := v1.ClusterRoleBindingArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name: pulumi.String("dogstatsd-standalone"),
		},
		RoleRef: v1.RoleRefArgs{
			ApiGroup: pulumi.String("rbac.authorization.k8s.io"),
			Kind:     pulumi.String("ClusterRole"),
			Name:     pulumi.String("dogstatsd-standalone"),
		},
		Subjects: v1.SubjectArray{
			&v1.SubjectArgs{
				Kind:      pulumi.String("ServiceAccount"),
				Name:      pulumi.String("dogstatsd-standalone"),
				Namespace: pulumi.String(namespace),
			},
		},
	}

	if _, err := corev1.NewServiceAccount(e.Ctx, "dogstatsd-standalone", &serviceAccountArgs, opts...); err != nil {
		return nil, err
	}

	if _, err := v1.NewClusterRole(e.Ctx, "dogstatsd-standalone", &clusterRoleArgs, opts...); err != nil {
		return nil, err
	}

	if _, err := v1.NewClusterRoleBinding(e.Ctx, "dogstatsd-standalone", &clusterRoleBindingArgs, opts...); err != nil {
		return nil, err
	}

	if _, err := appsv1.NewDaemonSet(e.Ctx, "dogstatsd-standalone", &daemonSetArgs, opts...); err != nil {
		return nil, err
	}

	return k8sComponent, nil
}
