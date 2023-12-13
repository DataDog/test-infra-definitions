package mutatedbyadmissioncontroller

import (
	"github.com/DataDog/test-infra-definitions/common/config"

	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type K8sComponent struct {
	pulumi.ResourceState
}

func K8sAppDefinition(e config.CommonEnvironment, kubeProvider *kubernetes.Provider, namespace string, opts ...pulumi.ResourceOption) (*K8sComponent, error) {
	opts = append(opts, pulumi.Provider(kubeProvider), pulumi.Parent(kubeProvider), pulumi.DeletedWith(kubeProvider))

	k8sComponent := &K8sComponent{}
	if err := e.Ctx.RegisterComponentResource("dd:apps", "mutated", k8sComponent, opts...); err != nil {
		return nil, err
	}

	opts = append(opts, pulumi.Parent(k8sComponent))

	ns, err := corev1.NewNamespace(e.Ctx, namespace, &corev1.NamespaceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name: pulumi.String(namespace),
		},
	}, opts...)
	if err != nil {
		return nil, err
	}

	opts = append(opts, pulumi.Parent(ns))

	if _, err := appsv1.NewDeployment(e.Ctx, "mutated", &appsv1.DeploymentArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("mutated"),
			Namespace: pulumi.String(namespace),
			Labels:    pulumi.StringMap{"app": pulumi.String("mutated")},
		},
		Spec: &appsv1.DeploymentSpecArgs{
			Replicas: pulumi.Int(1),
			Selector: &metav1.LabelSelectorArgs{
				MatchLabels: pulumi.StringMap{"app": pulumi.String("mutated")},
			},
			Template: &corev1.PodTemplateSpecArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Labels: pulumi.StringMap{
						"app":                             pulumi.String("mutated"),
						"admission.datadoghq.com/enabled": pulumi.String("true"),
						"tags.datadoghq.com/env":          pulumi.String("e2e"),
						"tags.datadoghq.com/service":      pulumi.String("mutated"),
						"tags.datadoghq.com/version":      pulumi.String("v0.0.1"),
					},
				},
				Spec: &corev1.PodSpecArgs{
					Containers: corev1.ContainerArray{
						corev1.ContainerArgs{
							Name:  pulumi.String("mutated"),
							Image: pulumi.String("busybox"),
							Command: pulumi.ToStringArray([]string{
								"/bin/sh",
								"-c",
							}),
							Args: pulumi.ToStringArray([]string{
								`printf 'DD_DOGSTATSD_URL:\t%s\n' "${DD_DOGSTATSD_URL:-❌ not set}";` +
									`printf 'DD_TRACE_AGENT_URL:\t%s\n' "${DD_TRACE_AGENT_URL:-❌ not set}";` +
									`printf 'DD_ENTITY_ID:\t%s\n' "${DD_ENTITY_ID:-❌ not set}";` +
									`printf 'DD_ENV:\t%s\n'       "${DD_ENV:-❌ not set}";` +
									`printf 'DD_SERVICE:\t%s\n'   "${DD_SERVICE:-❌ not set}";` +
									`printf 'DD_VERSION:\t%s\n'   "${DD_VERSION:-❌ not set}";` +
									`sleep infinity`,
							}),
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
