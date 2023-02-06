package resources

import (
	"strings"

	"github.com/pulumi/pulumi-libvirt/sdk/go/libvirt"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	DomainName    = "domainName"
	SharedFSMount = "sharedFSMount"
	DomainID      = "domainID"
	MACAddress    = "mac"
)

var kernelCmdlines = []map[string]interface{}{
	{"acpi": pulumi.String("off")},
	{"panic": pulumi.String("-1")},
	{"root": pulumi.String("/dev/vda")},
	{"net.ifnames": pulumi.String("0")},
	{"_": pulumi.String("rw")},
}

type ResourceCollection interface {
	GetDomainXLS(args map[string]interface{}) string
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

func formatResourceXML(xml string, args map[string]interface{}) string {
	var templateArgs []string
	for k, v := range args {
		templateArgs = append(templateArgs, "{"+k+"}", v.(string))
	}

	r := strings.NewReplacer(templateArgs...)
	return r.Replace(xml)
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
