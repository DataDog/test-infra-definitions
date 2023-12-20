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
func DescriptorFromString(descStr string) Descriptor {
	parts := strings.Split(descStr, osDescriptorSep)
	if len(parts) == 2 {
		return NewDescriptor(FlavorFromString(parts[0]), parts[1])
	} else if len(parts) == 3 {
		return NewDescriptorWithArch(FlavorFromString(parts[0]), parts[1], ArchitectureFromString(parts[2]))
	}

	panic(fmt.Sprintf("invalid OS descriptor string, was: %s", descStr))
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
