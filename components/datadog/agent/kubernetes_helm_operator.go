package agent

import (
	"dario.cat/mergo"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apiextensions"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	kubeHelm "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"gopkg.in/yaml.v3"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
	componentskube "github.com/DataDog/test-infra-definitions/components/kubernetes"
	"github.com/DataDog/test-infra-definitions/resources/helm"
)

// OperatorHelmInstallationArgs is the set of arguments for creating a new HelmInstallation component
type OperatorHelmInstallationArgs struct {
	// KubeProvider is the Kubernetes provider to use
	KubeProvider *kubernetes.Provider
	// Namespace is the namespace in which to install the agent
	Namespace string
	// ValuesYAML is used to provide installation-specific values
	ValuesYAML pulumi.AssetOrArchiveArrayInput
	// OperatorFullImagePath is used to specify the full image path for the cluster agent
	OperatorFullImagePath string
}

type OperatorHelmComponent struct {
	pulumi.ResourceState

	LinuxHelmReleaseName   pulumi.StringPtrOutput
	LinuxHelmReleaseStatus kubeHelm.ReleaseStatusOutput
}

func NewOperatorHelmInstallation(e config.Env, args OperatorHelmInstallationArgs, opts ...pulumi.ResourceOption) (*HelmComponent, error) {
	apiKey := e.AgentAPIKey()
	appKey := e.AgentAPPKey()
	baseName := "dd-operator"
	opts = append(opts, pulumi.Providers(args.KubeProvider), e.WithProviders(config.ProviderRandom), pulumi.Parent(args.KubeProvider), pulumi.DeletedWith(args.KubeProvider))

	helmComponent := &HelmComponent{}
	if err := e.Ctx().RegisterComponentResource("dd:agent-with-operator", "operator", helmComponent, opts...); err != nil {
		return nil, err
	}
	opts = append(opts, pulumi.Parent(helmComponent))

	// Create namespace if necessary
	ns, err := corev1.NewNamespace(e.Ctx(), args.Namespace, &corev1.NamespaceArgs{
		Metadata: metav1.ObjectMetaArgs{
			Name: pulumi.String(args.Namespace),
		},
	}, opts...)
	if err != nil {
		return nil, err
	}
	opts = append(opts, utils.PulumiDependsOn(ns))

	// Create secret if necessary
	secret, err := corev1.NewSecret(e.Ctx(), "datadog-credentials", &corev1.SecretArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: ns.Metadata.Name(),
			Name:      pulumi.Sprintf("%s-datadog-credentials", baseName),
		},
		StringData: pulumi.StringMap{
			"api-key": apiKey,
			"app-key": appKey,
		},
	}, opts...)
	if err != nil {
		return nil, err
	}
	opts = append(opts, utils.PulumiDependsOn(secret))

	// Create image pull secret if necessary
	var imgPullSecret *corev1.Secret
	if e.ImagePullRegistry() != "" {
		imgPullSecret, err = NewImagePullSecret(e, args.Namespace, opts...)
		if err != nil {
			return nil, err
		}
		opts = append(opts, utils.PulumiDependsOn(imgPullSecret))
	}

	operatorImagePath, operatorImageTag := utils.ParseImageReference(args.OperatorFullImagePath)
	linuxInstallName := baseName + "-linux"

	values := buildLinuxOperatorHelmValues(baseName, operatorImagePath, operatorImageTag)
	values.configureOperatorImagePullSecret(imgPullSecret)

	linux, err := helm.NewInstallation(e, helm.InstallArgs{
		RepoURL:     DatadogHelmRepo,
		ChartName:   "datadog-operator",
		InstallName: linuxInstallName,
		Namespace:   args.Namespace,
		ValuesYAML:  args.ValuesYAML,
		Values:      pulumi.Map(values),
	}, opts...)

	if err != nil {
		return nil, err
	}

	helmComponent.LinuxHelmReleaseName = linux.Name
	helmComponent.LinuxHelmReleaseStatus = linux.Status

	resourceOutputs := pulumi.Map{
		"linuxHelmReleaseName":   linux.Name,
		"linuxHelmReleaseStatus": linux.Status,
	}

	if err := e.Ctx().RegisterResourceOutputs(helmComponent, resourceOutputs); err != nil {
		return nil, err
	}

	return helmComponent, nil
}

func buildLinuxOperatorHelmValues(baseName string, operatorImagePath string, operatorImageTag string) HelmValues {
	return HelmValues{
		"apiKeyExistingSecret": pulumi.String(baseName + "-datadog-credentials"),
		"appKeyExistingSecret": pulumi.String(baseName + "-datadog-credentials"),
		"image": pulumi.Map{
			"repository": pulumi.String(operatorImagePath),
			"tag":        pulumi.String(operatorImageTag),
		},
		"logLevel": pulumi.String("debug"),
		"introspection": pulumi.Map{
			"enabled": pulumi.Bool(false),
		},
		"datadogAgentProfile": pulumi.Map{
			"enabled": pulumi.Bool(false),
		},
		"supportExtendedDaemonset": pulumi.Bool(false),
		"operatorMetricsEnabled":   pulumi.Bool(true),
		"metricsPort":              pulumi.Int(8383),
		"datadogAgent": pulumi.Map{
			"enabled": pulumi.Bool(true),
		},
		"datadogMonitor": pulumi.Map{
			"enabled": pulumi.Bool(false),
		},
		"datadogSLO": pulumi.Map{
			"enabled": pulumi.Bool(false),
		},
		"resources": pulumi.Map{
			"limits": pulumi.Map{
				"cpu":    pulumi.String("100m"),
				"memory": pulumi.String("250Mi"),
			},
			"requests": pulumi.Map{
				"cpu":    pulumi.String("100m"),
				"memory": pulumi.String("250Mi"),
			},
		},
		"installCRDs": pulumi.Bool(true),
		"datadogCRDs": pulumi.Map{
			"crds": pulumi.Map{
				"datadogAgents":   pulumi.Bool(true),
				"datadogMetrics":  pulumi.Bool(true),
				"datadogMonitors": pulumi.Bool(true),
				"datadogSLOs":     pulumi.Bool(true),
			},
		},
	}
}

func (values HelmValues) configureOperatorImagePullSecret(secret *corev1.Secret) {
	if secret == nil {
		return
	}

	values["imagePullSecrets"] = pulumi.MapArray{
		pulumi.Map{
			"name": secret.Metadata.Name(),
		},
	}

}

func K8sAppDefinition(e config.Env, kubeProvider *kubernetes.Provider, namespace string, fakeIntake *fakeintake.Fakeintake, kubeletTLSVerify bool, clusterName string, customDda string, opts ...pulumi.ResourceOption) (*componentskube.Workload, error) {
	opts = append(opts, pulumi.Provider(kubeProvider), pulumi.Parent(kubeProvider), pulumi.DeletedWith(kubeProvider))

	k8sComponent := &componentskube.Workload{}
	if err := e.Ctx().RegisterComponentResource("dd:agent-with-operator", "dda", k8sComponent, opts...); err != nil {
		return nil, err
	}

	opts = append(opts, pulumi.Parent(k8sComponent))

	ns, err := corev1.NewNamespace(
		e.Ctx(),
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

	var imagePullSecrets corev1.LocalObjectReferenceArray
	if e.ImagePullRegistry() != "" {
		imgPullSecret, err := NewImagePullSecret(e, namespace, opts...)
		if err != nil {
			return nil, err
		}

		imagePullSecrets = append(imagePullSecrets, corev1.LocalObjectReferenceArgs{
			Name: imgPullSecret.Metadata.Name(),
		})
	}

	ddaArgs := kubernetes.UntypedArgs{}
	err = yaml.Unmarshal([]byte(customDda), &ddaArgs)
	if err != nil {
		return nil, err
	}

	defaultArgs := kubernetes.UntypedArgs{
		"spec": pulumi.Map{
			"global": pulumi.Map{
				"credentials": pulumi.Map{
					"apiKey": pulumi.StringInput(e.AgentAPIKey()),
					"appKey": pulumi.StringInput(e.AgentAPPKey()),
				},
				"clusterName": pulumi.String(clusterName),
				"kubelet": pulumi.Map{
					"tlsVerify": pulumi.Bool(kubeletTLSVerify),
				},
			},
		},
	}

	if e.AgentUseFakeintake() {
		err = mergo.Merge(&ddaArgs, kubernetes.UntypedArgs{
			"spec": pulumi.Map{
				"override": pulumi.Map{
					"nodeAgent": pulumi.Map{
						"env": pulumi.MapArray{
							pulumi.Map{
								"name":  pulumi.String("DD_ADDITIONAL_ENDPOINTS"),
								"value": pulumi.Sprintf(`{"%s": ["FAKEAPIKEY"]}`, fakeIntake.URL),
							},
						},
					},
					"clusterAgent": pulumi.Map{
						"env": pulumi.MapArray{
							pulumi.Map{
								"name":  pulumi.String("DD_ADDITIONAL_ENDPOINTS"),
								"value": pulumi.Sprintf(`{"%s": ["FAKEAPIKEY"]}`, fakeIntake.URL),
							},
						},
					},
					"clusterChecksRunner": pulumi.Map{
						"env": pulumi.MapArray{
							pulumi.Map{
								"name":  pulumi.String("DD_ADDITIONAL_ENDPOINTS"),
								"value": pulumi.Sprintf(`{"%s": ["FAKEAPIKEY"]}`, fakeIntake.URL),
							},
						},
					},
				}},
		})
		if err != nil {
			return nil, err
		}
	}

	err = mergo.Merge(&ddaArgs, defaultArgs)
	if err != nil {
		return nil, err
	}

	_, err = apiextensions.NewCustomResource(e.Ctx(), "datadog-agent", &apiextensions.CustomResourceArgs{
		ApiVersion: pulumi.String("datadoghq.com/v2alpha1"),
		Kind:       pulumi.String("DatadogAgent"),
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("datadog"),
			Namespace: pulumi.String("datadog"),
		},
		OtherFields: ddaArgs,
	}, opts...)
	if err != nil {
		return nil, err
	}

	return k8sComponent, nil
}
