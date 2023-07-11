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

// String format is <flavor>:<version>(:<arch>)
func NewDescriptorFromString(descStr string) Descriptor {
	parts := strings.Split(descStr, osDescriptorSep)
	if len(parts) == 2 {
		return NewDescriptor(NewFlavorFromString(parts[0]), parts[1])
	} else if len(parts) == 3 {
		return NewDescriptorWithArch(NewFlavorFromString(parts[0]), parts[1], NewArchitectureFromString(parts[2]))
	} else {
		panic(fmt.Sprintf("invalid OS descriptor string, was: %s", descStr))
	}
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

func (d Descriptor) Family() Family {
	return d.family
}

func (d Descriptor) WithArch(a Architecture) Descriptor {
	d.Architecture = a
	return d
}
