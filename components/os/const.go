package os

import (
	"fmt"
	"strings"
)

type Architecture string

const (
	AMD64Arch = Architecture("x86_64")
	ARM64Arch = Architecture("arm64")
)

func NewArchitectureFromString(archStr string) Architecture {
	archStr = strings.ToLower(archStr)
	switch archStr {
	case "x86_64", "amd64", "": // Default architecture is AMD64
		return AMD64Arch
	case "arm64", "aarch64":
		return ARM64Arch
	default:
		panic(fmt.Sprintf("unknown architecture: %s", archStr))
	}
}

type Family int

const (
	UnknownFamily Family = iota

	LinuxFamily   Family = iota
	WindowsFamily Family = iota
	MacOSFamily   Family = iota
)

type Flavor int

const (
	Unknown Flavor = iota

	// Linux
	Ubuntu         Flavor = iota
	AmazonLinux    Flavor = iota
	AmazonLinuxECS Flavor = iota
	Debian         Flavor = iota
	RedHat         Flavor = iota
	Suse           Flavor = iota
	Fedora         Flavor = iota
	CentOS         Flavor = iota
	RockyLinux     Flavor = iota

	// Windows
	WindowsServer Flavor = 500

	// MacOS
	MacosOS Flavor = 1000
)

func NewFlavorFromString(flavorStr string) Flavor {
	flavorStr = strings.ToLower(flavorStr)
	switch flavorStr {
	case "", "ubuntu": // Default flavor is Ubuntu
		return Ubuntu
	case "amazon-linux":
		return AmazonLinux
	case "amazon-linux-ecs":
		return AmazonLinuxECS
	case "debian":
		return Debian
	case "redhat":
		return RedHat
	case "suse":
		return Suse
	case "fedora":
		return Fedora
	case "centos":
		return CentOS
	case "rocky-linux":
		return RockyLinux
	case "windows", "windows-server":
		return WindowsServer
	case "macos":
		return MacosOS
	default:
		panic(fmt.Sprintf("unknown OS flavor: %s", flavorStr))
	}
}

func (f Flavor) Type() Family {
	switch {
	case f < WindowsServer:
		return LinuxFamily
	case f < MacosOS:
		return WindowsFamily
	case f == MacosOS:
		return MacOSFamily
	default:
		panic("unknown OS flavor")
	}
}
