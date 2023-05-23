package os

import (
	"fmt"

	commonos "github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/resources/azure"
	"github.com/DataDog/test-infra-definitions/resources/azure/compute"
)

type Type int

const (
	WindowsOS Type = iota
	UbuntuOS       = iota
)

func GetOS(env azure.Environment, osType Type) (commonos.OS, error) {
	switch osType {
	case WindowsOS:
		return newWindows(env), nil
	case UbuntuOS:
		return newUbuntu(env), nil
	default:
		return nil, fmt.Errorf("cannot find environment: %v", osType)
	}
}

type ubuntu struct {
	commonos.Ubuntu
}

func newUbuntu(env azure.Environment) *ubuntu {
	return &ubuntu{
		Ubuntu: *commonos.NewUbuntu(&env),
	}
}

func (*ubuntu) GetSSHUser() string { return "azureuser" }

func (u *ubuntu) GetImage(arch commonos.Architecture) (string, error) {
	if arch != commonos.AMD64Arch {
		return "", fmt.Errorf("%v is not supported", arch)
	}
	return compute.UbuntuLatestURN(), nil
}

type windows struct {
	*commonos.Windows
}

func newWindows(env azure.Environment) *windows {
	return &windows{
		Windows: commonos.NewWindows(&env),
	}
}

func (*windows) GetSSHUser() string { return "azureuser" }

func (w *windows) GetImage(arch commonos.Architecture) (string, error) {
	if arch != commonos.AMD64Arch {
		return "", fmt.Errorf("%v is not supported", arch)
	}
	return compute.WindowsLatestURN(), nil
}
