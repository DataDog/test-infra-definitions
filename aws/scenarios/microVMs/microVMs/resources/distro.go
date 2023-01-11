package resources

import (
	_ "embed"

	"github.com/pulumi/pulumi-libvirt/sdk/go/libvirt"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type DistroResourceCollection struct {
	recipe string
}

func NewDistroResourceCollection(recipe string) *DistroResourceCollection {
	return &DistroResourceCollection{
		recipe: recipe,
	}
}

func (a *DistroResourceCollection) GetDomainXLS(args ...interface{}) string {
	return GetDefaultDomainXLS(args...)
}

func (a *DistroResourceCollection) GetNetworkXLS(args ...interface{}) string {
	return GetDefaultNetworkXLS(args...)
}

func (a *DistroResourceCollection) GetVolumeXML(args ...interface{}) string {
	return GetDefaultVolumeXML(args...)
}

func (a *DistroResourceCollection) GetPoolXML(args ...interface{}) string {
	return GetDefaultPoolXML(args...)
}

func (a *DistroResourceCollection) GetLibvirtDomainArgs(args *RecipeLibvirtDomainArgs) *libvirt.DomainArgs {
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
		Memory: pulumi.Int(args.Memory),
		Vcpu:   pulumi.Int(args.Vcpu),
		Xml: libvirt.DomainXmlArgs{
			Xslt: pulumi.String(args.Xls),
		},
	}
}
