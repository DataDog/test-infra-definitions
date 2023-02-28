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
	DHCPEntries   = "dhcpEntries"
	ImageName     = "imageName"
	VolumeKey     = "volumeKey"
	VolumePath    = "volumePath"
	PoolName      = "poolName"
	PoolPath      = "poolPath"
)

var kernelCmdlines = []map[string]interface{}{
	{"acpi": pulumi.String("off")},
	{"panic": pulumi.String("-1")},
	{"root": pulumi.String("/dev/vda")},
	{"net.ifnames": pulumi.String("0")},
	{"_": pulumi.String("rw")},
}

type ResourceCollection interface {
	GetDomainXLS(args map[string]pulumi.StringInput) pulumi.StringOutput
	GetNetworkXLS(args map[string]pulumi.StringInput) pulumi.StringOutput
	GetVolumeXML(args map[string]pulumi.StringInput) pulumi.StringOutput
	GetPoolXML(args map[string]pulumi.StringInput) pulumi.StringOutput
	GetLibvirtDomainArgs(*RecipeLibvirtDomainArgs) *libvirt.DomainArgs
}

type RecipeLibvirtDomainArgs struct {
	Vcpu              int
	Memory            int
	Xls               pulumi.StringOutput
	KernelPath        string
	Volume            *libvirt.Volume
	Resources         ResourceCollection
	ExtraKernelParams map[string]string
}

func formatResourceXML(xml string, args map[string]pulumi.StringInput) pulumi.StringOutput {
	var templateArgsPromise []interface{}

	// The Replacer functionality expects a list in the format
	// `{placeholder} val` as input for formatting a piece of text
	for k, v := range args {
		templateArgsPromise = append(templateArgsPromise, pulumi.Sprintf("{%s}", k), v)
	}

	pulumiXML := pulumi.All(templateArgsPromise...).ApplyT(func(promises []interface{}) (string, error) {
		var templateArgs []string

		for _, promise := range promises {
			templateArgs = append(templateArgs, promise.(string))
		}

		r := strings.NewReplacer(templateArgs...)
		return r.Replace(xml), nil
	}).(pulumi.StringOutput)

	return pulumiXML
}

func NewResourceCollection(recipe string) ResourceCollection {
	switch recipe {
	case "custom-arm64":
		return NewARM64ResourceCollection(recipe)
	case "custom-amd64":
		return NewAMD64ResourceCollection(recipe)
	case "distro-arm64":
		return NewDistroARM64ResourceCollection(recipe)
	case "distro-amd64":
		return NewDistroAMD64ResourceCollection(recipe)
	default:
		panic("unknown recipe: " + recipe)
	}
}
