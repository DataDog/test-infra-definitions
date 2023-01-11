package resources

import (
	"fmt"

	_ "embed"

	"github.com/pulumi/pulumi-libvirt/sdk/go/libvirt"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

//go:embed amd64/domain.xls
var amd64DomainXLS string

type AMD64ResourceCollection struct {
	recipe string
}

func NewAMD64ResourceCollection(recipe string) *AMD64ResourceCollection {
	return &AMD64ResourceCollection{
		recipe: recipe,
	}
}

func (a *AMD64ResourceCollection) GetDomainXLS(args ...interface{}) string {
	return fmt.Sprintf(amd64DomainXLS, args...)
}

func (a *AMD64ResourceCollection) GetNetworkXLS(args ...interface{}) string {
	return GetDefaultNetworkXLS(args...)
}

func (a *AMD64ResourceCollection) GetVolumeXML(args ...interface{}) string {
	return GetDefaultVolumeXML(args...)
}

func (a *AMD64ResourceCollection) GetPoolXML(args ...interface{}) string {
	return GetDefaultPoolXML(args...)
}

func (a *AMD64ResourceCollection) GetLibvirtDomainArgs(args *RecipeLibvirtDomainArgs) *libvirt.DomainArgs {
	return &libvirt.DomainArgs{
		Consoles: libvirt.DomainConsoleArray{
			libvirt.DomainConsoleArgs{
				Type:       pulumi.String("pty"),
				TargetPort: pulumi.String("0"),
				TargetType: pulumi.String("serial"),
			},
		},
		Disks: libvirt.DomainDiskArray{
			libvirt.DomainDiskArgs{
				VolumeId: args.Volume.ID(),
			},
		},
		Kernel: pulumi.String(args.KernelPath),
		Cmdlines: pulumi.MapArray{
			pulumi.Map{"console": pulumi.String("ttyS0")},
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
