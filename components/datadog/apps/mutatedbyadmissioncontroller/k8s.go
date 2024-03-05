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

// K8sAppDefinitions creates a Kubernetes deployment annotated with a specific
// lib version and another one without the annotation.
func K8sAppDefinitions(e config.CommonEnvironment, kubeProvider *kubernetes.Provider, namespace string, opts ...pulumi.ResourceOption) (*K8sComponent, error) {
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

	if err = k8sDeployment(e, namespace, true, opts...); err != nil {
		return nil, err
	}

	if err = k8sDeployment(e, namespace, false, opts...); err != nil {
		return nil, err
	}

	return k8sComponent, nil
}

func k8sDeployment(e config.CommonEnvironment, namespace string, withLibInjectionAnnotation bool, opts ...pulumi.ResourceOption) error {
	name := "mutated"
	annotations := pulumi.StringMap{}
	if withLibInjectionAnnotation {
		name = "mutated-with-lib-annotation"
		annotations["admission.datadoghq.com/python-lib.version"] = pulumi.String("v2.7.0")
	}

	if _, err := appsv1.NewDeployment(e.Ctx, name, &appsv1.DeploymentArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(name),
			Namespace: pulumi.String(namespace),
			Labels:    pulumi.StringMap{"app": pulumi.String(name)},
		},
		Spec: &appsv1.DeploymentSpecArgs{
			Replicas: pulumi.Int(1),
			Selector: &metav1.LabelSelectorArgs{
				MatchLabels: pulumi.StringMap{"app": pulumi.String(name)},
			},
			Template: &corev1.PodTemplateSpecArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Labels: pulumi.StringMap{
						"app":                             pulumi.String(name),
						"admission.datadoghq.com/enabled": pulumi.String("true"),
						"tags.datadoghq.com/env":          pulumi.String("e2e"),
						"tags.datadoghq.com/service":      pulumi.String("mutated"),
						"tags.datadoghq.com/version":      pulumi.String("v0.0.1"),
					},
					Annotations: annotations,
				},
				Spec: &corev1.PodSpecArgs{
					Containers: corev1.ContainerArray{
						corev1.ContainerArgs{
							Name:  pulumi.String(name),
							Image: pulumi.String("python:3.12-slim"),
							Command: pulumi.ToStringArray([]string{
								"python", "-c", "while True: import time; time.sleep(60)",
							}),
						},
					},
				},
			},
		},
	}, opts...); err != nil {
		return err
	}

	return nil
}
