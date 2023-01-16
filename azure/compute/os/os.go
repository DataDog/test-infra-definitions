package os

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/azure"
	"github.com/DataDog/test-infra-definitions/azure/compute"
	"github.com/DataDog/test-infra-definitions/common/os"
)

type Type int

const (
	WindowsOS Type = iota
	UbuntuOS       = iota
)

func GetOS(env azure.Environment, osType Type) (os.OS, error) {
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
	os.Ubuntu
}

func newUbuntu(env azure.Environment) *ubuntu {
	return &ubuntu{
		Ubuntu: *os.NewUbuntu(&env),
	}
}

func (*ubuntu) GetSSHUser() string { return "azureuser" }

func (u *ubuntu) GetImage(arch os.Architecture) (string, error) {
	if arch != os.AMD64Arch {
		return "", fmt.Errorf("%v is not supported", arch)
	}
	return compute.UbuntuLatestURN(), nil
}

type windows struct {
	*os.Windows
}

func newWindows(env azure.Environment) *windows {
	return &windows{
		Windows: os.NewWindows(&env),
	}
}

func (*windows) GetSSHUser() string { return "azureuser" }

func (w *windows) GetImage(arch os.Architecture) (string, error) {
	if arch != os.AMD64Arch {
		return "", fmt.Errorf("%v is not supported", arch)
	}
	return compute.WindowsLatestURN(), nil
}
