package fakeintake

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/components"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	v1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const localPort = 30080

func NewLocalKubernetesFakeintake(e config.CommonEnvironment, resourceName string, kubeProvider *kubernetes.Provider) (*Fakeintake, error) {
	return components.NewComponent(e, resourceName, func(comp *Fakeintake) error {
		v1.NewDeployment(e.Ctx, resourceName, &v1.DeploymentArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.String(resourceName),
			},
			Spec: &v1.DeploymentSpecArgs{
				Replicas: pulumi.Int(1),
				Selector: &metav1.LabelSelectorArgs{
					MatchLabels: pulumi.StringMap{
						"app": pulumi.String(resourceName),
					},
				},
				Template: corev1.PodTemplateSpecArgs{
					Metadata: &metav1.ObjectMetaArgs{
						Labels: pulumi.StringMap{
							"app": pulumi.String(resourceName),
						},
					},
					Spec: &corev1.PodSpecArgs{
						Containers: corev1.ContainerArray{
							&corev1.ContainerArgs{
								Name:  pulumi.String(resourceName),
								Image: pulumi.String("public.ecr.aws/datadog/fakeintake:latest"),
								ReadinessProbe: &corev1.ProbeArgs{
									HttpGet: &corev1.HTTPGetActionArgs{
										Path: pulumi.String("/fakeintake/health"),
										Port: pulumi.Int(80),
									},
								},
							},
						},
					},
				},
			},
		}, pulumi.Provider(kubeProvider))

		corev1.NewService(e.Ctx, resourceName, &corev1.ServiceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.String(resourceName),
			},
			Spec: &corev1.ServiceSpecArgs{
				Selector: pulumi.StringMap{
					"app": pulumi.String(resourceName),
				},
				Ports: corev1.ServicePortArray{
					&corev1.ServicePortArgs{
						NodePort:   pulumi.Int(localPort),
						Port:       pulumi.Int(80),
						TargetPort: pulumi.Int(80),
					},
				},
				Type: pulumi.String("NodePort"),
			},
		}, pulumi.Provider(kubeProvider))

		comp.Port = 80
		comp.Scheme = "http"
		comp.Host = pulumi.Sprintf("%s.default.svc.cluster.local", resourceName)
		comp.URL = pulumi.Sprintf("%s://%s:%d", comp.Scheme, comp.Host, comp.Port)
		comp.ClientURL = pulumi.Sprintf("http://localhost:%v", localPort)

		return nil

	})
}
