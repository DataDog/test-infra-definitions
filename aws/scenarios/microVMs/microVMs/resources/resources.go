package resources

import (
	_ "embed"
	"os"
	"path/filepath"
)

//go:embed default/domain.xls
var defaultDomainXLS string

//go:embed default/network.xls
var defaultNetworkXLS string

//go:embed default/pool.xml
var defaultPoolXML string

//go:embed default/volume.xml
var defaultVolumeXML string

func GetRecipeDomainTemplate(recipe string) (string, error) {
	domainXLSFile := filepath.Join("microVMs", "resources", recipe, "domain.xls")
	domainXLS, err := os.ReadFile(domainXLSFile)
	if err != nil {
		return "", err
	}

	return string(domainXLS), nil
}

func GetRecipeDomainTemplateOrDefault(recipe string) string {
	domainXLS, err := GetRecipeDomainTemplate(recipe)
	if err != nil {
		return defaultDomainXLS
	}

	return domainXLS
}

func GetRecipeNetworkTemplate(recipe string) (string, error) {
	networkXLSFile := filepath.Join("microVMs", "resources", recipe, "network.xls")
	networkXLS, err := os.ReadFile(networkXLSFile)
	if err != nil {
		return "", err
	}

	return string(networkXLS), nil
}

func GetRecipeNetworkTemplateOrDefault(recipe string) string {
	networkXLS, err := GetRecipeNetworkTemplate(recipe)
	if err != nil {
		return defaultNetworkXLS
	}

	return networkXLS
}

func GetRecipePoolTemplate(recipe string) (string, error) {
	poolXMLFile := filepath.Join("microVMs", "resources", recipe, "pool.xml")
	poolXML, err := os.ReadFile(poolXMLFile)
	if err != nil {
		return "", err
	}

	return string(poolXML), nil
}

func GetRecipePoolTemplateOrDefault(recipe string) string {
	poolXML, err := GetRecipePoolTemplate(recipe)
	if err != nil {
		return defaultPoolXML
	}

	return poolXML
}

func GetRecipeVolumeTemplate(recipe string) (string, error) {
	volumeXMLFile := filepath.Join("microVMs", "resources", recipe, "volume.xml")
	volumeXML, err := os.ReadFile(volumeXMLFile)
	if err != nil {
		return "", err
	}

	return string(volumeXML), nil
}

func GetRecipeVolumeTemplateOrDefault(recipe string) string {
	volumeXML, err := GetRecipeVolumeTemplate(recipe)
	if err != nil {
		return defaultVolumeXML
	}

	return volumeXML
}
