package cilium

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/resources/helm"

	kubeHelm "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type HelmComponent struct {
	pulumi.ResourceState

	CiliumBaseHelmReleaseStatus kubeHelm.ReleaseStatusOutput
}

func NewHelmInstallation(e config.Env, opts ...pulumi.ResourceOption) (*HelmComponent, error) {
	helmComponent := &HelmComponent{}
	if err := e.Ctx().RegisterComponentResource("dd:cilium", "cilium", helmComponent, opts...); err != nil {
		return nil, err
	}
	opts = append(opts, pulumi.Parent(helmComponent))
	values := buildCiliumBaseHelmValues()

	ciliumBase, err := helm.NewInstallation(e, helm.InstallArgs{
		RepoURL:     "https://helm.cilium.io",
		ChartName:   "cilium",
		InstallName: "cilium",
		Namespace:   "kube-system",
		Values:      pulumi.Map(values),
		Version:     pulumi.StringPtr("1.17.0"),
	}, opts...)
	if err != nil {
		return nil, err
	}

	helmComponent.CiliumBaseHelmReleaseStatus = ciliumBase.Status

	opts = append(opts, utils.PulumiDependsOn(ciliumBase))
	resourceOutputs := pulumi.Map{
		"CiliumBaseHelmReleaseStatus": ciliumBase.Status,
	}

	if err := e.Ctx().RegisterResourceOutputs(helmComponent, resourceOutputs); err != nil {
		return nil, err
	}

	return helmComponent, nil
}

type HelmValues pulumi.Map

func buildCiliumBaseHelmValues() HelmValues {
	return HelmValues{
		"kubeProxyReplacement": pulumi.BoolPtr(true),
	}
}
