package resources

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/pulumi/pulumi-libvirt/sdk/go/libvirt"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/DataDog/test-infra-definitions/scenarios/aws/microVMs/vmconfig"
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

const (
	fileConsole = "file"
	ptyConsole  = "pty"
)

const (
	RAMPool     vmconfig.PoolType = "ram"
	DefaultPool vmconfig.PoolType = "default"
)

var consoles = map[string]libvirt.DomainConsoleArgs{
	fileConsole: {
		Type:       pulumi.String("file"),
		TargetPort: pulumi.String("0"),
		TargetType: pulumi.String("serial"),
	},
	ptyConsole: {
		Type:       pulumi.String("pty"),
		TargetPort: pulumi.String("0"),
		TargetType: pulumi.String("serial"),
	},
}

var kernelCmdlines = []map[string]interface{}{
	{"acpi": pulumi.String("off")},
	{"panic": pulumi.String("-1")},
	{"root": pulumi.String("/dev/vda")},
	{"net.ifnames": pulumi.String("0")},
	{"_": pulumi.String("rw")},
}

type ResourceCollection interface {
	GetDomainXLS(args map[string]pulumi.StringInput) pulumi.StringOutput
	GetVolumeXML(*RecipeLibvirtVolumeArgs) pulumi.StringOutput
	GetPoolXML(args map[string]pulumi.StringInput) pulumi.StringOutput
	GetLibvirtDomainArgs(*RecipeLibvirtDomainArgs) (*libvirt.DomainArgs, error)
}

type AttachMethod int

const (
	AttachAsFile AttachMethod = iota
	AttachAsVolume
)

type DiskTarget string

type DomainDisk struct {
	VolumeID   pulumi.StringPtrInput
	Attach     AttachMethod
	Target     string
	Mountpoint string
}

type RecipeLibvirtDomainArgs struct {
	DomainName        string
	Vcpu              int
	Memory            int
	Xls               pulumi.StringOutput
	KernelPath        string
	Disks             []DomainDisk
	Resources         ResourceCollection
	ExtraKernelParams map[string]string
	Machine           string
	ConsoleType       string
}

type RecipeLibvirtVolumeArgs struct {
	PoolType vmconfig.PoolType
	XMLArgs  map[string]pulumi.StringInput
}

func setupConsole(consoleType, domainName string) (libvirt.DomainConsoleArgs, error) {
	if consoleType == fileConsole {
		fname := fmt.Sprintf("/var/log/libvirt/ddvm-%s.log", domainName)
		_ = os.Remove(fname)
		f, err := os.Create(fname)
		if err != nil {
			return libvirt.DomainConsoleArgs{}, fmt.Errorf("failed to create console output file %s: %v", fname, err)
		}
		defer f.Close()

		console := consoles[consoleType]
		console.SourcePath = pulumi.String(fname)
		return console, nil
	}

	// default console type is `pty`
	return consoles[ptyConsole], nil
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
