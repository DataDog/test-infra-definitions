package resources

import (
	_ "embed"
	"fmt"
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

func GetDefaultNetworkXLS(args ...interface{}) string {
	return fmt.Sprintf(defaultNetworkXLS, args...)
}

func GetDefaultVolumeXML(args ...interface{}) string {
	return fmt.Sprintf(defaultVolumeXML, args...)
}

func GetDefaultPoolXML(args ...interface{}) string {
	return fmt.Sprintf(defaultPoolXML, args...)
}
