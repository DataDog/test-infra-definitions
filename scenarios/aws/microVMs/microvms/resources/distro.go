package resources

import (
	// import embed
	_ "embed"
	"fmt"
	"sort"

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

func (a *DistroAMD64ResourceCollection) GetVolumeXML(args *RecipeLibvirtVolumeArgs) pulumi.StringOutput {
	return GetDefaultVolumeXML(args, a.recipe)
}

func (a *DistroAMD64ResourceCollection) GetPoolXML(args map[string]pulumi.StringInput) pulumi.StringOutput {
	return GetDefaultPoolXML(args, a.recipe)
}

func (a *DistroAMD64ResourceCollection) GetLibvirtDomainArgs(args *RecipeLibvirtDomainArgs) (*libvirt.DomainArgs, error) {
	var disks libvirt.DomainDiskArray
	sort.Slice(args.Disks, func(i, j int) bool {
		return args.Disks[i].Target < args.Disks[j].Target
	})
	for _, disk := range args.Disks {
		switch disk.Attach {
		case AttachAsFile:
			disks = append(disks, libvirt.DomainDiskArgs{
				File: disk.VolumeID,
			})
		case AttachAsVolume:
			disks = append(disks, libvirt.DomainDiskArgs{
				VolumeId: disk.VolumeID,
			})
		default:
		}
	}

	console, err := setupConsole(args.ConsoleType, args.DomainName)
	if err != nil {
		return nil, fmt.Errorf("failed to setup console for domain %s: %v", args.DomainName, err)
	}

	domainArgs := libvirt.DomainArgs{
		Name: pulumi.String(args.DomainName),
		Consoles: libvirt.DomainConsoleArray{
			console,
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

	return &domainArgs, nil
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

func (a *DistroARM64ResourceCollection) GetVolumeXML(args *RecipeLibvirtVolumeArgs) pulumi.StringOutput {
	return GetDefaultVolumeXML(args, a.recipe)
}

func (a *DistroARM64ResourceCollection) GetPoolXML(args map[string]pulumi.StringInput) pulumi.StringOutput {
	return GetDefaultPoolXML(args, a.recipe)
}

func (a *DistroARM64ResourceCollection) GetLibvirtDomainArgs(args *RecipeLibvirtDomainArgs) (*libvirt.DomainArgs, error) {
	var disks libvirt.DomainDiskArray

	sort.Slice(args.Disks, func(i, j int) bool {
		return args.Disks[i].Target < args.Disks[j].Target
	})
	for _, disk := range args.Disks {
		switch disk.Attach {
		case AttachAsFile:
			disks = append(disks, libvirt.DomainDiskArgs{
				File: disk.VolumeID,
			})
		case AttachAsVolume:
			disks = append(disks, libvirt.DomainDiskArgs{
				VolumeId: disk.VolumeID,
			})
		default:
		}
	}

	console, err := setupConsole(args.ConsoleType, args.DomainName)
	if err != nil {
		return nil, fmt.Errorf("failed to setup console for domain %s: %v", args.DomainName, err)
	}

	domainArgs := libvirt.DomainArgs{
		Name: pulumi.String(args.DomainName),
		Consoles: libvirt.DomainConsoleArray{
			console,
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

	return &domainArgs, nil
}
