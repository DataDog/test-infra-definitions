package etcd

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	componentskube "github.com/DataDog/test-infra-definitions/components/kubernetes"
)

// This stores an openmetrics check configuration in etcd.
// It runs the check against the example prometheus app already deployed
// ("apps-prometheus").
// The check renames one of the prometheus metrics so that we can verify that
// the check was discovered.
const setupCommands = `
# Start etcd in background
etcd --enable-v2 \
     --name=etcd-0 \
     --data-dir=/var/lib/etcd \
     --listen-client-urls=http://0.0.0.0:2379 \
     --advertise-client-urls=http://etcd:2379 \
     --listen-peer-urls=http://0.0.0.0:2380 \
     --initial-advertise-peer-urls=http://etcd:2380 \
     --initial-cluster=etcd-0=http://etcd:2380 \
     --initial-cluster-token=etcd-cluster-1 \
     --initial-cluster-state=new &

# Wait for etcd to be ready
until etcdctl endpoint health; do
  echo "Waiting for etcd..."
  sleep 1
done

# It's not always immediately ready. Let's wait a bit more to be sure.
sleep 10

export ETCDCTL_API=2
etcdctl set /datadog/check_configs/apps-prometheus/check_names '["openmetrics"]'
etcdctl set /datadog/check_configs/apps-prometheus/init_configs '[{}]'
etcdctl set /datadog/check_configs/apps-prometheus/instances '[{"openmetrics_endpoint": "http://%%host%%:8080/metrics", "metrics":[{"prom_gauge": "prom_gauge_configured_in_etcd"}]}]'

wait
`

func K8sAppDefinition(e config.Env, kubeProvider *kubernetes.Provider, namespace string, opts ...pulumi.ResourceOption) (*componentskube.Workload, error) {
	opts = append(opts, pulumi.Provider(kubeProvider), pulumi.Parent(kubeProvider), pulumi.DeletedWith(kubeProvider))

	k8sComponent := &componentskube.Workload{}
	if err := e.Ctx().RegisterComponentResource("dd:apps", "etcd", k8sComponent, opts...); err != nil {
		return nil, err
	}

	opts = append(opts, pulumi.Parent(k8sComponent))

	ns, err := corev1.NewNamespace(e.Ctx(), namespace, &corev1.NamespaceArgs{
		Metadata: metav1.ObjectMetaArgs{
			Name: pulumi.String(namespace),
		},
	}, opts...)
	if err != nil {
		return nil, err
	}

	opts = append(opts, utils.PulumiDependsOn(ns))

	_, err = appsv1.NewDeployment(e.Ctx(), "etcd", &appsv1.DeploymentArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("etcd"),
			Namespace: pulumi.String(namespace),
			Labels: pulumi.StringMap{
				"app": pulumi.String("etcd"),
			},
		},
		Spec: &appsv1.DeploymentSpecArgs{
			Replicas: pulumi.Int(1),
			Selector: &metav1.LabelSelectorArgs{
				MatchLabels: pulumi.StringMap{
					"app": pulumi.String("etcd"),
				},
			},
			Template: &corev1.PodTemplateSpecArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Labels: pulumi.StringMap{
						"app": pulumi.String("etcd"),
					},
				},
				Spec: &corev1.PodSpecArgs{
					Containers: corev1.ContainerArray{
						&corev1.ContainerArgs{
							Name: pulumi.String("etcd"),
							// The agent only supports the v2 API, which is not
							// supported anymore in newer versions of etcd.
							Image: pulumi.String("quay.io/coreos/etcd:v3.5.1"),
							Command: pulumi.StringArray{
								pulumi.String("/bin/sh"),
								pulumi.String("-c"),
							},
							Args: pulumi.StringArray{
								pulumi.String(setupCommands),
							},
							Ports: &corev1.ContainerPortArray{
								&corev1.ContainerPortArgs{
									Name:          pulumi.String("etcd"),
									ContainerPort: pulumi.Int(2379),
									Protocol:      pulumi.String("TCP"),
								},
							},
							ReadinessProbe: &corev1.ProbeArgs{
								HttpGet: &corev1.HTTPGetActionArgs{
									Path:   pulumi.String("/health"),
									Port:   pulumi.Int(2379),
									Scheme: pulumi.String("HTTP"),
								},
								InitialDelaySeconds: pulumi.Int(10),
								TimeoutSeconds:      pulumi.Int(5),
							},
							LivenessProbe: &corev1.ProbeArgs{
								HttpGet: &corev1.HTTPGetActionArgs{
									Path:   pulumi.String("/health"),
									Port:   pulumi.Int(2379),
									Scheme: pulumi.String("HTTP"),
								},
								InitialDelaySeconds: pulumi.Int(10),
								TimeoutSeconds:      pulumi.Int(5),
							},
						},
					},
				},
			},
		},
	}, opts...)
	if err != nil {
		return nil, err
	}

	_, err = corev1.NewService(e.Ctx(), "etcd", &corev1.ServiceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("etcd"),
			Namespace: pulumi.String(namespace),
			Labels: pulumi.StringMap{
				"app": pulumi.String("etcd"),
			},
		},
		Spec: &corev1.ServiceSpecArgs{
			Selector: pulumi.StringMap{
				"app": pulumi.String("etcd"),
			},
			Ports: &corev1.ServicePortArray{
				&corev1.ServicePortArgs{
					Name:       pulumi.String("client"),
					Port:       pulumi.Int(2379),
					TargetPort: pulumi.Int(2379),
					Protocol:   pulumi.String("TCP"),
				},
			},
		},
	}, opts...)
	if err != nil {
		return nil, err
	}

	return k8sComponent, nil
}
