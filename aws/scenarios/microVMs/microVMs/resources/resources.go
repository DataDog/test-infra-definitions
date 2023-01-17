package resources

import (
	_ "embed"

	"github.com/pulumi/pulumi-libvirt/sdk/go/libvirt"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var kernelCmdlines = []map[string]interface{}{
	map[string]interface{}{"acpi": pulumi.String("off")},
	map[string]interface{}{"panic": pulumi.String("-1")},
	map[string]interface{}{"root": pulumi.String("/dev/vda")},
	map[string]interface{}{"net.ifnames": pulumi.String("0")},
	map[string]interface{}{"_": pulumi.String("rw")},
}

type ResourceCollection interface {
	GetDomainXLS(args ...interface{}) string
	GetNetworkXLS(args ...interface{}) string
	GetVolumeXML(args ...interface{}) string
	GetPoolXML(args ...interface{}) string
	GetLibvirtDomainArgs(*RecipeLibvirtDomainArgs) *libvirt.DomainArgs
}

type RecipeLibvirtDomainArgs struct {
	Vcpu              int
	Memory            int
	Xls               string
	KernelPath        string
	Volume            *libvirt.Volume
	Resources         ResourceCollection
	ExtraKernelParams map[string]string
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
