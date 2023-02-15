package resources

import (
	// import embed
	_ "embed"

	"github.com/pulumi/pulumi-libvirt/sdk/go/libvirt"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

//go:embed distro/domain.xls
var distroDomainXLS string

type DistroResourceCollection struct {
	recipe string
}

func NewDistroResourceCollection(recipe string) *DistroResourceCollection {
	return &DistroResourceCollection{
		recipe: recipe,
	}
}

func (a *DistroResourceCollection) GetDomainXLS(args map[string]pulumi.StringInput) pulumi.StringOutput {
	return formatResourceXML(distroDomainXLS, args)
}

func (a *DistroResourceCollection) GetNetworkXLS(args map[string]pulumi.StringInput) pulumi.StringOutput {
	return GetDefaultNetworkXLS(args)
}

func (a *DistroResourceCollection) GetVolumeXML(args map[string]pulumi.StringInput) pulumi.StringOutput {
	return GetDefaultVolumeXML(args)
}

func (a *DistroResourceCollection) GetPoolXML(args map[string]pulumi.StringInput) pulumi.StringOutput {
	return GetDefaultPoolXML(args)
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
			Xslt: args.Xls,
		},
	}
}
