package cilium

import (
	"reflect"

	kubeHelm "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/DataDog/test-infra-definitions/common"
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/resources/helm"
)

type HelmValues pulumi.Map

type Params struct {
	HelmValues HelmValues
}

type Option = func(*Params) error

func NewParams(options ...Option) (*Params, error) {
	return common.ApplyOption(&Params{}, options)
}

func WithHelmValues(values HelmValues) Option {
	return func(p *Params) error {
		p.HelmValues = values
		return nil
	}
}

type HelmComponent struct {
	pulumi.ResourceState

	CiliumHelmReleaseStatus kubeHelm.ReleaseStatusOutput
}

func boolValue(i pulumi.Input) bool {
	pv := reflect.ValueOf(i)
	if pv.Kind() == reflect.Ptr {
		if pv.IsNil() {
			return false
		}

		pv = pv.Elem()
	}

	return pv.Bool()
}

func (p *Params) HasKubeProxyReplacement() bool {
	if v, ok := p.HelmValues["kubeProxyReplacement"]; ok {
		return boolValue(v)
	}

	return false
}

func NewHelmInstallation(e config.Env, params *Params, opts ...pulumi.ResourceOption) (*HelmComponent, error) {
	helmComponent := &HelmComponent{}
	if err := e.Ctx().RegisterComponentResource("dd:cilium", "cilium", helmComponent, opts...); err != nil {
		return nil, err
	}
	opts = append(opts, pulumi.Parent(helmComponent))
	ciliumBase, err := helm.NewInstallation(e, helm.InstallArgs{
		RepoURL:     "https://helm.cilium.io",
		ChartName:   "cilium",
		InstallName: "cilium",
		Namespace:   "kube-system",
		Values:      pulumi.Map(params.HelmValues),
		Version:     pulumi.StringPtr("1.17.0"),
	}, opts...)
	if err != nil {
		return nil, err
	}

	helmComponent.CiliumHelmReleaseStatus = ciliumBase.Status

	opts = append(opts, utils.PulumiDependsOn(ciliumBase))
	resourceOutputs := pulumi.Map{
		"CiliumBaseHelmReleaseStatus": ciliumBase.Status,
	}

	if err := e.Ctx().RegisterResourceOutputs(helmComponent, resourceOutputs); err != nil {
		return nil, err
	}

	return helmComponent, nil
}
