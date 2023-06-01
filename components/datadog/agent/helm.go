package agent

import (
	"fmt"

	"golang.org/x/exp/maps"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/resources/helm"

	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	kubeHelm "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/helm/v3"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	DatadogHelmRepo = "https://helm.datadoghq.com"
)

type HelmInstallationArgs struct {
	KubeProvider  *kubernetes.Provider
	Namespace     string
	ValuesYAML    pulumi.AssetOrArchiveArrayInput
	DeployWindows bool
}

type HelmComponent struct {
	pulumi.ResourceState

	LinuxHelmReleaseName   pulumi.StringPtrOutput
	LinuxHelmReleaseStatus kubeHelm.ReleaseStatusOutput

	WindowsHelmReleaseName   pulumi.StringPtrOutput
	WindowsHelmReleaseStatus kubeHelm.ReleaseStatusOutput
}

func NewHelmInstallation(e config.CommonEnvironment, args HelmInstallationArgs, opts ...pulumi.ResourceOption) (*HelmComponent, error) {
	apiKey := e.AgentAPIKey()
	appKey := e.AgentAPPKey()
	installName := "dda"
	opts = append(opts, pulumi.Provider(args.KubeProvider), pulumi.Parent(args.KubeProvider), pulumi.DeletedWith(args.KubeProvider))

	helmComponent := &HelmComponent{}
	if err := e.Ctx.RegisterComponentResource("dd:agent", "dda", helmComponent, opts...); err != nil {
		return nil, err
	}

	opts = append(opts, pulumi.Parent(helmComponent))

	// Create namespace if necessary
	ns, err := corev1.NewNamespace(e.Ctx, args.Namespace, &corev1.NamespaceArgs{
		Metadata: metav1.ObjectMetaArgs{
			Name: pulumi.String(args.Namespace),
		},
	}, opts...)
	if err != nil {
		return nil, err
	}

	opts = append(opts, utils.PulumiDependsOn(ns))

	// Create secret if necessary
	secret, err := corev1.NewSecret(e.Ctx, "datadog-credentials", &corev1.SecretArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: ns.Metadata.Name(),
			Name:      pulumi.String(fmt.Sprintf("%s-datadog-credentials", installName)),
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

	// Compute some values
	agentImagePath := DockerAgentFullImagePath(&e, "")
	agentImagePath, agentImageTag := utils.ParseImageReference(agentImagePath)

	clusterAgentImagePath := DockerClusterAgentFullImagePath(&e, "")
	clusterAgentImagePath, clusterAgentImageTag := utils.ParseImageReference(clusterAgentImagePath)

	linuxInstallName := installName
	if args.DeployWindows {
		linuxInstallName += "-linux"
	}

	linux, err := helm.NewInstallation(e, helm.InstallArgs{
		RepoURL:     DatadogHelmRepo,
		ChartName:   "datadog",
		InstallName: linuxInstallName,
		Namespace:   args.Namespace,
		ValuesYAML:  args.ValuesYAML,
		Values:      buildLinuxHelmValues(installName, agentImagePath, agentImageTag, clusterAgentImagePath, clusterAgentImageTag),
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

	if args.DeployWindows {
		windows, err := helm.NewInstallation(e, helm.InstallArgs{
			RepoURL:     DatadogHelmRepo,
			ChartName:   "datadog",
			InstallName: installName + "-windows",
			Namespace:   args.Namespace,
			ValuesYAML:  args.ValuesYAML,
			Values:      buildWindowsHelmValues(installName, agentImagePath, agentImageTag, clusterAgentImagePath, clusterAgentImageTag),
		}, opts...)
		if err != nil {
			return nil, err
		}

		helmComponent.WindowsHelmReleaseName = windows.Name
		helmComponent.WindowsHelmReleaseStatus = windows.Status

		maps.Copy(resourceOutputs, pulumi.Map{
			"windowsHelmReleaseName":   windows.Name,
			"windowsHelmReleaseStatus": windows.Status,
		})
	}

	if err := e.Ctx.RegisterResourceOutputs(helmComponent, resourceOutputs); err != nil {
		return nil, err
	}

	return helmComponent, nil
}

func buildLinuxHelmValues(installName string, agentImagePath, agentImageTag, clusterAgentImagePath, clusterAgentImageTag string) pulumi.Map {
	return pulumi.Map{
		"datadog": pulumi.Map{
			"apiKeyExistingSecret": pulumi.String(installName + "-datadog-credentials"),
			"appKeyExistingSecret": pulumi.String(installName + "-datadog-credentials"),
			"checksCardinality":    pulumi.String("high"),
			"logs": pulumi.Map{
				"enabled":             pulumi.Bool(true),
				"containerCollectAll": pulumi.Bool(true),
			},
			"dogstatsd": pulumi.Map{
				"originDetection": pulumi.Bool(true),
				"tagCardinality":  pulumi.String("high"),
				"useHostPort":     pulumi.Bool(true),
			},
			"apm": pulumi.Map{
				"portEnabled": pulumi.Bool(true),
			},
			"processAgent": pulumi.Map{
				"processCollection": pulumi.Bool(true),
			},
			"helmCheck": pulumi.Map{
				"enabled": pulumi.Bool(true),
			},
		},
		"agents": pulumi.Map{
			"image": pulumi.Map{
				"repository":    pulumi.String(agentImagePath),
				"tag":           pulumi.String(agentImageTag),
				"doNotCheckTag": pulumi.Bool(true),
			},
		},
		"clusterAgent": pulumi.Map{
			"enabled": pulumi.Bool(true),
			"image": pulumi.Map{
				"repository":    pulumi.String(clusterAgentImagePath),
				"tag":           pulumi.String(clusterAgentImageTag),
				"doNotCheckTag": pulumi.Bool(true),
			},
			"metricsProvider": pulumi.Map{
				"enabled":           pulumi.Bool(true),
				"useDatadogMetrics": pulumi.Bool(true),
			},
		},
		"clusterChecksRunner": pulumi.Map{
			"enabled": pulumi.Bool(true),
			"image": pulumi.Map{
				"repository":    pulumi.String(agentImagePath),
				"tag":           pulumi.String(agentImageTag),
				"doNotCheckTag": pulumi.Bool(true),
			},
		},
	}
}

func buildWindowsHelmValues(installName string, agentImagePath, agentImageTag, _, _ string) pulumi.Map {
	return pulumi.Map{
		"targetSystem": pulumi.String("windows"),
		"datadog": pulumi.Map{
			"apiKeyExistingSecret": pulumi.String(installName + "-datadog-credentials"),
			"appKeyExistingSecret": pulumi.String(installName + "-datadog-credentials"),
			"checksCardinality":    pulumi.String("high"),
			"logs": pulumi.Map{
				"enabled":             pulumi.Bool(true),
				"containerCollectAll": pulumi.Bool(true),
			},
			"dogstatsd": pulumi.Map{
				"originDetection": pulumi.Bool(true),
				"tagCardinality":  pulumi.String("high"),
				"useHostPort":     pulumi.Bool(true),
			},
			"apm": pulumi.Map{
				"portEnabled": pulumi.Bool(true),
			},
			"processAgent": pulumi.Map{
				"processCollection": pulumi.Bool(true),
			},
		},
		"agents": pulumi.Map{
			"image": pulumi.Map{
				"repository":    pulumi.String(agentImagePath),
				"tag":           pulumi.String(agentImageTag),
				"doNotCheckTag": pulumi.Bool(true),
			},
		},
		// Make the Windows node agents target the Linux cluster agent
		"clusterAgent": pulumi.Map{
			"enabled": pulumi.Bool(false),
		},
		"existingClusterAgent": pulumi.Map{
			"join":                 pulumi.Bool(true),
			"serviceName":          pulumi.String(installName + "-linux-datadog-cluster-agent"),
			"tokenSecretName":      pulumi.String(installName + "-linux-datadog-cluster-agent"),
			"clusterchecksEnabled": pulumi.Bool(false),
		},
	}
}
