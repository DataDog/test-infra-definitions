package helm

import (
	"github.com/DataDog/test-infra-definitions/common/config"

	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func NewInstallation(e config.CommonEnvironment, kubeProvider *kubernetes.Provider, repoURL, chartName, installName, namespace string, inlineValues pulumi.MapInput, valueFiles []string, opts ...pulumi.ResourceOption) (*helm.Release, error) {
	valueAssets := make(pulumi.AssetOrArchiveArray, 0, len(valueFiles))
	for _, valuePath := range valueFiles {
		valueAssets = append(valueAssets, pulumi.NewFileAsset(valuePath))
	}

	opts = append(opts, pulumi.Provider(kubeProvider))
	return helm.NewRelease(e.Ctx, installName, &helm.ReleaseArgs{
		Namespace: pulumi.StringPtr(namespace),
		Name:      pulumi.StringPtr(installName),
		RepositoryOpts: helm.RepositoryOptsArgs{
			Repo: pulumi.StringPtr(repoURL),
		},
		Chart:            pulumi.String(chartName),
		CreateNamespace:  pulumi.BoolPtr(true),
		DependencyUpdate: pulumi.BoolPtr(true),
		ValueYamlFiles:   valueAssets,
		Values:           inlineValues,
	}, opts...)
}
