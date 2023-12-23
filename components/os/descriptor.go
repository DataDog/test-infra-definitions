package os

import (
	"fmt"
	"strings"
)

const osDescriptorSep = ":"

// Descriptor provides definition of an OS
type Descriptor struct {
	family       Family
	Flavor       Flavor
	Version      string
	Architecture Architecture
}

func NewDescriptor(f Flavor, version string) Descriptor {
	return NewDescriptorWithArch(f, version, AMD64Arch)
}

func NewDescriptorWithArch(f Flavor, version string, arch Architecture) Descriptor {
	return Descriptor{
		family:       f.Type(),
		Flavor:       f,
		Version:      version,
		Architecture: arch,
	}
}

// String format is <flavor>:<version>(:<arch>)
func DescriptorFromString(descStr string, defaultFlavor Flavor) Descriptor {
	parts := strings.Split(descStr, osDescriptorSep)
	if len(parts) < 2 || len(parts) > 3 {
		panic(fmt.Sprintf("invalid OS descriptor string, was: %s", descStr))
	}

	var flavor Flavor
	if parts[0] == "" {
		flavor = defaultFlavor
	} else {
		flavor = FlavorFromString(parts[0])
	}

	if len(parts) == 3 {
		return NewDescriptorWithArch(flavor, parts[1], ArchitectureFromString(parts[2]))
	}

	return NewDescriptor(flavor, parts[1])
}

func (d Descriptor) Family() Family {
	return d.family
}

func (d Descriptor) WithArch(a Architecture) Descriptor {
	d.Architecture = a
	return d
}

func (d Descriptor) String() string {
	return strings.Join([]string{d.Flavor.String(), d.Version, string(d.Architecture)}, osDescriptorSep)
}
