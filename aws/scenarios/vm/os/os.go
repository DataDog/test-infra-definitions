package os

import (
	"github.com/DataDog/test-infra-definitions/aws"
)

type Architecture string

const (
	AMD64Arch = Architecture("x86_64")
	ARM64Arch = Architecture("arm64")
)

type OSType int

const (
	WindowsOS OSType = iota
	UbuntuOS         = iota
	MacOS            = iota
)

type OS interface {
	GetSSHUser() string
	GetAMI(aws.Environment, Architecture) (string, error)
	GetAMIArch(arch Architecture) string
	GetDefaultInstanceType(arch Architecture) string
	GetServiceManager() *serviceManager
	GetTenancy() string
	GetConfigPath() string
	GetOSType() OSType
}

func GetOS(os OSType) OS {
	switch os {
	case WindowsOS:
		return windows{}
	case UbuntuOS:
		return ubuntu{}
	case MacOS:
		return macOS{}
	default:
		panic("OS not supported")
	}
}
