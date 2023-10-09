package resources

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/DataDog/test-infra-definitions/scenarios/aws/microVMs/vmconfig"
	"github.com/pulumi/pulumi-libvirt/sdk/go/libvirt"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	SharedFSMount = "sharedFSMount"
	DomainID      = "domainID"
	MACAddress    = "mac"
	DHCPEntries   = "dhcpEntries"
	ImageName     = "imageName"
	VolumeKey     = "volumeKey"
	ImagePath     = "imagePath"
	PoolName      = "poolName"
	PoolPath      = "poolPath"
	Nvram         = "nvram"
	Efi           = "efi"
	Format        = "format"
	VCPU          = "vcpu"
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
	GetVolumeXML(args map[string]pulumi.StringInput) pulumi.StringOutput
	GetPoolXML(args map[string]pulumi.StringInput) pulumi.StringOutput
	GetLibvirtDomainArgs(*RecipeLibvirtDomainArgs) *libvirt.DomainArgs
}

type RecipeLibvirtDomainArgs struct {
	DomainName        string
	Vcpu              int
	Memory            int
	Xls               pulumi.StringOutput
	KernelPath        string
	Volumes           []*libvirt.Volume
	Resources         ResourceCollection
	ExtraKernelParams map[string]string
	Machine           string
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

func isLocalRecipe(recipe string) bool {
	return (recipe == vmconfig.RecipeCustomLocal) || (recipe == vmconfig.RecipeDistroLocal)
}

func GetLocalArchRecipe(recipe string) string {
	var prefix string

	if !isLocalRecipe(recipe) {
		return recipe
	}

	if strings.HasPrefix(recipe, "distro") {
		prefix = "distro"
	} else if strings.HasPrefix(recipe, "custom") {
		prefix = "custom"
	} else {
		panic("unknown recipe " + recipe)
	}

	if runtime.GOARCH == "amd64" {
		return fmt.Sprintf("%s-x86_64", prefix)
	} else if runtime.GOARCH == "arm64" {
		return fmt.Sprintf("%s-arm64", prefix)
	}

	panic("unknown recipe " + recipe)
}

func NewResourceCollection(recipe string) ResourceCollection {
	archSpecificRecipe := GetLocalArchRecipe(recipe)

	switch archSpecificRecipe {
	case vmconfig.RecipeCustomARM64:
		return NewARM64ResourceCollection(recipe)
	case vmconfig.RecipeCustomAMD64:
		return NewAMD64ResourceCollection(recipe)
	case vmconfig.RecipeDistroARM64:
		return NewDistroARM64ResourceCollection(recipe)
	case vmconfig.RecipeDistroAMD64:
		return NewDistroAMD64ResourceCollection(recipe)
	case vmconfig.RecipeDefault:
		return NewDefaultResourceCollection(recipe)
	default:
		panic("unknown recipe: " + archSpecificRecipe)
	}
}
