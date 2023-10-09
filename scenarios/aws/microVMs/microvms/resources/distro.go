package resources

import (
	// import embed
	_ "embed"

	"github.com/pulumi/pulumi-libvirt/sdk/go/libvirt"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

//go:embed distro/domain-amd64.xls
var distroDomainXLS string

//go:embed distro/domain-arm64.xls
var distroARM64DomainXLS string

type DistroAMD64ResourceCollection struct {
	recipe string
}

func NewDistroAMD64ResourceCollection(recipe string) *DistroAMD64ResourceCollection {
	return &DistroAMD64ResourceCollection{
		recipe: recipe,
	}
}

func (a *DistroAMD64ResourceCollection) GetDomainXLS(args map[string]pulumi.StringInput) pulumi.StringOutput {
	return formatResourceXML(distroDomainXLS, args)
}

func (a *DistroAMD64ResourceCollection) GetVolumeXML(args map[string]pulumi.StringInput) pulumi.StringOutput {
	return GetDefaultVolumeXML(args, a.recipe)
}

func (a *DistroAMD64ResourceCollection) GetPoolXML(args map[string]pulumi.StringInput) pulumi.StringOutput {
	return GetDefaultPoolXML(args, a.recipe)
}

func (a *DistroAMD64ResourceCollection) GetLibvirtDomainArgs(args *RecipeLibvirtDomainArgs) *libvirt.DomainArgs {
	var disks libvirt.DomainDiskArray
	for _, vol := range args.Volumes {
		disks = append(disks, libvirt.DomainDiskArgs{
			VolumeId: vol.ID(),
		})
	}

	domainArgs := libvirt.DomainArgs{
		Name: pulumi.String(args.DomainName),
		Consoles: libvirt.DomainConsoleArray{
			libvirt.DomainConsoleArgs{
				Type:       pulumi.String("pty"),
				TargetPort: pulumi.String("0"),
				TargetType: pulumi.String("serial"),
			},
		},
		Disks:  disks,
		Memory: pulumi.Int(args.Memory),
		Vcpu:   pulumi.Int(args.Vcpu),
		Xml: libvirt.DomainXmlArgs{
			Xslt: args.Xls,
		},
	}

	if args.Machine != "" {
		domainArgs.Machine = pulumi.String(args.Machine)
	}

	return &domainArgs
}

type DistroARM64ResourceCollection struct {
	recipe string
}

func NewDistroARM64ResourceCollection(recipe string) *DistroARM64ResourceCollection {
	return &DistroARM64ResourceCollection{
		recipe: recipe,
	}
}

func (a *DistroARM64ResourceCollection) GetDomainXLS(args map[string]pulumi.StringInput) pulumi.StringOutput {
	return formatResourceXML(distroARM64DomainXLS, args)
}

func (a *DistroARM64ResourceCollection) GetVolumeXML(args map[string]pulumi.StringInput) pulumi.StringOutput {
	return GetDefaultVolumeXML(args, a.recipe)
}

func (a *DistroARM64ResourceCollection) GetPoolXML(args map[string]pulumi.StringInput) pulumi.StringOutput {
	return GetDefaultPoolXML(args, a.recipe)
}

func (a *DistroARM64ResourceCollection) GetLibvirtDomainArgs(args *RecipeLibvirtDomainArgs) *libvirt.DomainArgs {
	var disks libvirt.DomainDiskArray
	for _, vol := range args.Volumes {
		disks = append(disks, libvirt.DomainDiskArgs{
			VolumeId: vol.ID(),
		})
	}

	domainArgs := libvirt.DomainArgs{
		Name: pulumi.String(args.DomainName),
		Consoles: libvirt.DomainConsoleArray{
			libvirt.DomainConsoleArgs{
				Type:       pulumi.String("pty"),
				TargetPort: pulumi.String("0"),
				TargetType: pulumi.String("serial"),
			},
		},
		Disks:   disks,
		Memory:  pulumi.Int(args.Memory),
		Vcpu:    pulumi.Int(args.Vcpu),
		Machine: pulumi.String("virt"),
		Xml: libvirt.DomainXmlArgs{
			Xslt: args.Xls,
		},
	}

	if args.Machine != "" {
		domainArgs.Machine = pulumi.String(args.Machine)
	}

	return &domainArgs
}
