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

//go:embed default/volume.xml
var defaultVolumeXML string

func GetDefaultDomainXLS(args ...interface{}) string {
	return defaultDomainXLS
}

func GetDefaultNetworkXLS(args map[string]pulumi.StringInput) pulumi.StringOutput {
	return formatResourceXML(defaultNetworkXLS, args)
}

func GetDefaultVolumeXML(args map[string]pulumi.StringInput) pulumi.StringOutput {
	return formatResourceXML(defaultVolumeXML, args)
}

func GetDefaultPoolXML(args map[string]pulumi.StringInput) pulumi.StringOutput {
	return formatResourceXML(defaultPoolXML, args)
}
