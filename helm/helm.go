package helm

import (
	"github.com/DataDog/test-infra-definitions/common/config"

	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type InstallArgs struct {
	KubernetesProvider *kubernetes.Provider
	RepoURL            string
	ChartName          string
	InstallName        string
	Namespace          string
	ValuesFilePaths    []string
	Values             pulumi.Map
}

func NewInstallation(e config.CommonEnvironment, args InstallArgs, opts ...pulumi.ResourceOption) (*helm.Release, error) {
	valueAssets := make(pulumi.AssetOrArchiveArray, 0, len(args.ValuesFilePaths))
	for _, valuePath := range args.ValuesFilePaths {
		valueAssets = append(valueAssets, pulumi.NewFileAsset(valuePath))
	}

	opts = append(opts, pulumi.Provider(args.KubernetesProvider))
	return helm.NewRelease(e.Ctx, args.InstallName, &helm.ReleaseArgs{
		Namespace: pulumi.StringPtr(args.Namespace),
		Name:      pulumi.StringPtr(args.InstallName),
		RepositoryOpts: helm.RepositoryOptsArgs{
			Repo: pulumi.StringPtr(args.RepoURL),
		},
		Chart:            pulumi.String(args.ChartName),
		CreateNamespace:  pulumi.BoolPtr(true),
		DependencyUpdate: pulumi.BoolPtr(true),
		ValueYamlFiles:   valueAssets,
		Values:           args.Values,
	}, opts...)
}
