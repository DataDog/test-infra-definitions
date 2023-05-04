package resources

import (
	// import embed
	_ "embed"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

//go:embed default/domain.xls
var defaultDomainXLS string

//go:embed default/network.xls
var defaultNetworkXLS string

//go:embed default/pool.xml
var defaultPoolXML string

//go:embed default/pool_local.xls
var defaultLocalPoolXLS string

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

func GetDefaultPoolXML(args map[string]pulumi.StringInput, recipe string) pulumi.StringOutput {
	if isLocalRecipe(recipe) {
		return formatResourceXML(defaultLocalPoolXLS, args)
	}

	return formatResourceXML(defaultPoolXML, args)
}
