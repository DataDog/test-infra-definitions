package os

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/aws"
	commonos "github.com/DataDog/test-infra-definitions/common/os"
)

type OS interface {
	commonos.OS
	GetAMIArch(arch commonos.Architecture) string
	GetTenancy() string
}

type Type int

const (
	WindowsOS Type = iota
	UbuntuOS       = iota
	MacosOS        = iota
)

func GetOS(env aws.Environment, osType Type) (OS, error) {
	switch osType {
	case WindowsOS:
		return newWindows(env), nil
	case UbuntuOS:
		return newUbuntu(env), nil
	case MacosOS:
		return newMacOS(env), nil
	default:
		return nil, fmt.Errorf("cannot find environment: %v", osType)
	}
}
