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
