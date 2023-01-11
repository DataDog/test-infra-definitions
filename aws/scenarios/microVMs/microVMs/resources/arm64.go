package resources

import (
	"fmt"

	_ "embed"

	"github.com/pulumi/pulumi-libvirt/sdk/go/libvirt"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

//go:embed arm64/domain.xls
var arm64DomainXLS string

type ARM64ResourceCollection struct {
	recipe string
}

func NewARM64ResourceCollection(recipe string) *ARM64ResourceCollection {
	return &ARM64ResourceCollection{
		recipe: recipe,
	}
}

func (a *ARM64ResourceCollection) GetDomainXLS(args ...interface{}) string {
	return fmt.Sprintf(arm64DomainXLS, args...)
}

func (a *ARM64ResourceCollection) GetNetworkXLS(args ...interface{}) string {
	return GetDefaultNetworkXLS(args...)
}

func (a *ARM64ResourceCollection) GetVolumeXML(args ...interface{}) string {
	return GetDefaultVolumeXML(args...)
}

func (a *ARM64ResourceCollection) GetPoolXML(args ...interface{}) string {
	return GetDefaultPoolXML(args...)
}

func (a *ARM64ResourceCollection) GetLibvirtDomainArgs(args *RecipeLibvirtDomainArgs) *libvirt.DomainArgs {
	return &libvirt.DomainArgs{
		Disks: libvirt.DomainDiskArray{
			libvirt.DomainDiskArgs{
				VolumeId: args.Volume.ID(),
			},
		},
		Machine: pulumi.String("virt"),
		Kernel:  pulumi.String(args.KernelPath),
		Cmdlines: pulumi.MapArray{
			pulumi.Map{"acpi": pulumi.String("off")},
			pulumi.Map{"panic": pulumi.String("-1")},
			pulumi.Map{"root": pulumi.String("/dev/vda")},
			pulumi.Map{"net.ifnames": pulumi.String("0")},
			pulumi.Map{"_": pulumi.String("rw")},
		},
		Memory: pulumi.Int(args.Memory),
		Vcpu:   pulumi.Int(args.Vcpu),
		Xml: libvirt.DomainXmlArgs{
			Xslt: pulumi.String(args.Xls),
		},
	}
}
