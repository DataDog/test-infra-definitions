package nginx

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps"
	componentskube "github.com/DataDog/test-infra-definitions/components/kubernetes"

	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// K8sAppDefinitionOpenShift creates nginx deployments for OpenShift using non-privileged ports
func K8sAppDefinitionOpenShift(e config.Env, kubeProvider *kubernetes.Provider, namespace string, opts ...pulumi.ResourceOption) (*componentskube.Workload, error) {
	opts = append(opts, pulumi.Provider(kubeProvider), pulumi.Parent(kubeProvider), pulumi.DeletedWith(kubeProvider))

	k8sComponent := &componentskube.Workload{}
	if err := e.Ctx().RegisterComponentResource("dd:apps", fmt.Sprintf("%s-nginx-openshift", namespace), k8sComponent, opts...); err != nil {
		return nil, err
	}

	opts = append(opts, pulumi.Parent(k8sComponent))

	ns, err := corev1.NewNamespace(e.Ctx(), namespace, &corev1.NamespaceArgs{
		Metadata: metav1.ObjectMetaArgs{
			Name: pulumi.String(namespace),
			Labels: pulumi.StringMap{
				"related_team": pulumi.String("contp"),
				"related_org":  pulumi.String("agent-org"),
			},
			Annotations: pulumi.StringMap{
				"related_email": pulumi.String("team-container-platform@datadoghq.com"),
			},
		},
	}, opts...)
	if err != nil {
		return nil, err
	}

	opts = append(opts, utils.PulumiDependsOn(ns))

	// Create ConfigMap with nginx configuration for port 8080
	nginxConfig := `worker_processes  auto;

events {
    worker_connections  4096;
}

http {
    server {
        listen [::]:8080 ipv6only=off reuseport fastopen=32 default_server;

        location /nginx_status {
          stub_status on;
          access_log  /dev/stdout;
          allow all;
        }
    }
}`

	if _, err := corev1.NewConfigMap(e.Ctx(), fmt.Sprintf("%s/nginx-config", namespace), &corev1.ConfigMapArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("nginx-config"),
			Namespace: pulumi.String(namespace),
		},
		Data: pulumi.StringMap{
			"nginx.conf": pulumi.String(nginxConfig),
		},
	}, opts...); err != nil {
		return nil, err
	}

	// Create nginx deployment with port 8080 instead of 80 for OpenShift compatibility
	if _, err := appsv1.NewDeployment(e.Ctx(), fmt.Sprintf("%s/nginx", namespace), &appsv1.DeploymentArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("nginx"),
			Namespace: pulumi.String(namespace),
			Annotations: pulumi.StringMap{
				"x-sub-team": pulumi.String("contint"),
			},
			Labels: pulumi.StringMap{
				"app":    pulumi.String("nginx"),
				"x-team": pulumi.String("contp"),
			},
		},
		Spec: &appsv1.DeploymentSpecArgs{
			Replicas: pulumi.Int(1),
			Selector: &metav1.LabelSelectorArgs{
				MatchLabels: pulumi.StringMap{
					"app": pulumi.String("nginx"),
				},
			},
			Template: &corev1.PodTemplateSpecArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Labels: pulumi.StringMap{
						"app":           pulumi.String("nginx"),
						"x-parent-type": pulumi.String("deployment"),
					},
					Annotations: pulumi.StringMap{
						"x-parent-name": pulumi.String("nginx"),
						"ad.datadoghq.com/nginx.checks": pulumi.String(utils.JSONMustMarshal(
							map[string]interface{}{
								"nginx": map[string]interface{}{
									"init_config":           map[string]interface{}{},
									"check_tag_cardinality": "high",
									"instances": []map[string]interface{}{
										{
											"nginx_status_url": "http://%%host%%:8080/nginx_status",
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
							Image: pulumi.String("ghcr.io/datadog/apps-nginx-server:" + apps.Version),
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
									ContainerPort: pulumi.Int(8080),
									Protocol:      pulumi.String("TCP"),
								},
							},
							LivenessProbe: &corev1.ProbeArgs{
								HttpGet: &corev1.HTTPGetActionArgs{
									Port: pulumi.Int(8080),
								},
								TimeoutSeconds: pulumi.Int(5),
							},
							ReadinessProbe: &corev1.ProbeArgs{
								HttpGet: &corev1.HTTPGetActionArgs{
									Port: pulumi.Int(8080),
								},
								TimeoutSeconds: pulumi.Int(5),
							},
							VolumeMounts: &corev1.VolumeMountArray{
								&corev1.VolumeMountArgs{
									Name:      pulumi.String("nginx-config"),
									MountPath: pulumi.String("/etc/nginx/nginx.conf"),
									SubPath:   pulumi.String("nginx.conf"),
								},
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
							Name: pulumi.String("nginx-config"),
							ConfigMap: &corev1.ConfigMapVolumeSourceArgs{
								Name: pulumi.String("nginx-config"),
							},
						},
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

	// Create service for nginx
	if _, err := corev1.NewService(e.Ctx(), fmt.Sprintf("%s/nginx", namespace), &corev1.ServiceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("nginx"),
			Namespace: pulumi.String(namespace),
			Labels: pulumi.StringMap{
				"app": pulumi.String("nginx"),
			},
			Annotations: pulumi.StringMap{
				"ad.datadoghq.com/service.checks": pulumi.String(utils.JSONMustMarshal(
					map[string]interface{}{
						"http_check": map[string]interface{}{
							"init_config": map[string]interface{}{},
							"instances": []map[string]interface{}{
								{
									"name":    "My Nginx",
									"url":     "http://%%host%%:8080",
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
					Port:       pulumi.Int(8080),
					TargetPort: pulumi.String("http"),
					Protocol:   pulumi.String("TCP"),
				},
			},
		},
	}, opts...); err != nil {
		return nil, err
	}

	// Create nginx query deployment
	if _, err := appsv1.NewDeployment(e.Ctx(), fmt.Sprintf("%s/nginx-query", namespace), &appsv1.DeploymentArgs{
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
							Image: pulumi.String("ghcr.io/datadog/apps-http-client:" + apps.Version),
							Args: pulumi.StringArray{
								pulumi.String("-url"),
								pulumi.String("http://nginx:8080"),
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
									"memory": pulumi.String("64Mi"),
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
