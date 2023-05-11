package resources

import (
	// import embed
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

func (a *AMD64ResourceCollection) GetDomainXLS(args map[string]pulumi.StringInput) pulumi.StringOutput {
	return formatResourceXML(amd64DomainXLS, args)
}

func (a *AMD64ResourceCollection) GetVolumeXML(args map[string]pulumi.StringInput) pulumi.StringOutput {
	return GetDefaultVolumeXML(args, a.recipe)
}

func (a *AMD64ResourceCollection) GetPoolXML(args map[string]pulumi.StringInput) pulumi.StringOutput {
	return GetDefaultPoolXML(args, a.recipe)
}

func (a *AMD64ResourceCollection) GetLibvirtDomainArgs(args *RecipeLibvirtDomainArgs) *libvirt.DomainArgs {
	var cmdlines []map[string]interface{}

	for cmd, val := range args.ExtraKernelParams {
		cmdlines = append(cmdlines, map[string]interface{}{cmd: pulumi.String(val)})
	}

	cmdlines = append(cmdlines, kernelCmdlines...)

	domainArgs := libvirt.DomainArgs{
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
		Kernel:   pulumi.String(args.KernelPath),
		Cmdlines: pulumi.ToMapArray(cmdlines),
		Memory:   pulumi.Int(args.Memory),
		Vcpu:     pulumi.Int(args.Vcpu),
		Xml: libvirt.DomainXmlArgs{
			Xslt: args.Xls,
		},
	}

	if args.Machine != "" {
		domainArgs.Machine = pulumi.String(args.Machine)
	}

	return &domainArgs
}
