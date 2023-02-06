package resources

import (
	// import embed
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

func (a *ARM64ResourceCollection) GetDomainXLS(args map[string]interface{}) string {
	return formatResourceXML(arm64DomainXLS, args)
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
	var cmdlines []map[string]interface{}
	for cmd, val := range args.ExtraKernelParams {
		cmdlines = append(cmdlines, map[string]interface{}{cmd: pulumi.String(val)})
	}

	cmdlines = append(cmdlines, kernelCmdlines...)

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
		Machine:  pulumi.String("virt"),
		Kernel:   pulumi.String(args.KernelPath),
		Cmdlines: pulumi.ToMapArray(cmdlines),
		Memory:   pulumi.Int(args.Memory),
		Vcpu:     pulumi.Int(args.Vcpu),
		Xml: libvirt.DomainXmlArgs{
			Xslt: pulumi.String(args.Xls),
		},
	}
}
