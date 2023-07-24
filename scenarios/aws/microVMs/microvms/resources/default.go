package resources

import (
	// import embed
	_ "embed"

	"github.com/pulumi/pulumi-libvirt/sdk/go/libvirt"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

//go:embed default/domain.xls
var defaultDomainXLS string

//go:embed default/network.xls
var defaultNetworkXLS string

//go:embed default/pool.xml
var defaultPoolXML string

//go:embed default/volume.xml
var defaultVolumeXML string

//go:embed default/volume_local.xls
var defaultLocalVolumeXLS string

func GetDefaultDomainXLS(...interface{}) string {
	return defaultDomainXLS
}

func GetDefaultNetworkXLS(args map[string]pulumi.StringInput) pulumi.StringOutput {
	return formatResourceXML(defaultNetworkXLS, args)
}

func GetDefaultVolumeXML(args map[string]pulumi.StringInput, recipe string) pulumi.StringOutput {
	if isLocalRecipe(recipe) {
		return formatResourceXML(defaultLocalVolumeXLS, args)
	}

	return formatResourceXML(defaultVolumeXML, args)
}

func GetDefaultPoolXML(args map[string]pulumi.StringInput, _ string) pulumi.StringOutput {
	return formatResourceXML(defaultPoolXML, args)
}

type DefaultResourceCollection struct {
	recipe string
}

func NewDefaultResourceCollection(recipe string) *DefaultResourceCollection {
	return &DefaultResourceCollection{
		recipe: recipe,
	}
}

func (a *DefaultResourceCollection) GetDomainXLS(_ map[string]pulumi.StringInput) pulumi.StringOutput {
	return pulumi.Sprintf("%s", GetDefaultDomainXLS())
}

func (a *DefaultResourceCollection) GetVolumeXML(args map[string]pulumi.StringInput) pulumi.StringOutput {
	return GetDefaultVolumeXML(args, a.recipe)
}

func (a *DefaultResourceCollection) GetPoolXML(args map[string]pulumi.StringInput) pulumi.StringOutput {
	return GetDefaultPoolXML(args, a.recipe)
}

func (a *DefaultResourceCollection) GetLibvirtDomainArgs(_ *RecipeLibvirtDomainArgs) *libvirt.DomainArgs {
	return nil
}
