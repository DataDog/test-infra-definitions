package resources

import (
	_ "embed"

	"github.com/pulumi/pulumi-libvirt/sdk/go/libvirt"
)

type ResourceCollection interface {
	GetDomainXLS(args ...interface{}) string
	GetNetworkXLS(args ...interface{}) string
	GetVolumeXML(args ...interface{}) string
	GetPoolXML(args ...interface{}) string
	GetLibvirtDomainArgs(*RecipeLibvirtDomainArgs) *libvirt.DomainArgs
}

type RecipeLibvirtDomainArgs struct {
	Vcpu       int
	Memory     int
	Xls        string
	KernelPath string
	Volume     *libvirt.Volume
	Resources  ResourceCollection
}

func NewResourceCollection(recipe string) ResourceCollection {
	if recipe == "custom-arm64" {
		return NewARM64ResourceCollection(recipe)
	} else if recipe == "custom-amd64" {
		return NewAMD64ResourceCollection(recipe)
	} else if recipe == "distro" {
		return NewDistroResourceCollection(recipe)
	}

	panic("unknown recipe: " + recipe)
}

//var recipeResources = map[string]func(string) ResourceCollection{
//	"custom-arm64": arm64Resources.NewARM64ResourceCollection,
//	"custom-amd64": amd64Resources.NewAMD64ResourceCollection,
//}
//
//func NewResourceCollection(recipe string) ResourceCollection {
//	fn, ok := recipeResources[recipe]
//	if !ok {
//		panic("unknown recpie")
//	}
//
//	return fn(recipe)
//}

///func GetRecipeDomainTemplate(recipe string) (string, error) {
///	domainXLSFile := filepath.Join("microVMs", "resources", recipe, "domain.xls")
///	domainXLS, err := os.ReadFile(domainXLSFile)
///	if err != nil {
///		return "", err
///	}
///
///	return string(domainXLS), nil
///}
///
///func GetRecipeDomainTemplateOrDefault(recipe string) string {
///	domainXLS, err := GetRecipeDomainTemplate(recipe)
///	if err != nil {
///		return defaultDomainXLS
///	}
///
///	return domainXLS
///}
///
///func GetRecipeNetworkTemplate(recipe string) (string, error) {
///	networkXLSFile := filepath.Join("microVMs", "resources", recipe, "network.xls")
///	networkXLS, err := os.ReadFile(networkXLSFile)
///	if err != nil {
///		return "", err
///	}
///
///	return string(networkXLS), nil
///}
///
///func GetRecipeNetworkTemplateOrDefault(recipe string) string {
///	networkXLS, err := GetRecipeNetworkTemplate(recipe)
///	if err != nil {
///		return defaultNetworkXLS
///	}
///
///	return networkXLS
///}
///
///func GetRecipePoolTemplate(recipe string) (string, error) {
///	poolXMLFile := filepath.Join("microVMs", "resources", recipe, "pool.xml")
///	poolXML, err := os.ReadFile(poolXMLFile)
///	if err != nil {
///		return "", err
///	}
///
///	return string(poolXML), nil
///}
///
///func GetRecipePoolTemplateOrDefault(recipe string) string {
///	poolXML, err := GetRecipePoolTemplate(recipe)
///	if err != nil {
///		return defaultPoolXML
///	}
///
///	return poolXML
///}
///
///func GetRecipeVolumeTemplate(recipe string) (string, error) {
///	volumeXMLFile := filepath.Join("microVMs", "resources", recipe, "volume.xml")
///	volumeXML, err := os.ReadFile(volumeXMLFile)
///	if err != nil {
///		return "", err
///	}
///
///	return string(volumeXML), nil
///}
///
///func GetRecipeVolumeTemplateOrDefault(recipe string) string {
///	volumeXML, err := GetRecipeVolumeTemplate(recipe)
///	if err != nil {
///		return defaultVolumeXML
///	}
///
///	return volumeXML
///}
