package os

import (
	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/common/os"
)

type OS interface {
	os.OS
	GetSSHUser() string
	GetAMIArch(arch os.Architecture) string
	GetTenancy() string
}

func GetOS(env aws.Environment, osType os.OSType) OS {
	switch osType {
	case os.WindowsOS:
		return newWindows(env)
	case os.UbuntuOS:
		return newUbuntu(env)
	case os.MacosOS:
		return newMacOS(env)
	default:
		panic("OS not supported")
	}
}
