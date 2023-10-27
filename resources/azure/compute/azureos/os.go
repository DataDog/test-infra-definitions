package azureos

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/resources/azure"
	"github.com/DataDog/test-infra-definitions/resources/azure/compute"
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
	env config.Environment
	os.Ubuntu
}

func newUbuntu(env azure.Environment) *ubuntu {
	return &ubuntu{
		env:    &env,
		Ubuntu: *os.NewUbuntu(),
	}
}

func (*ubuntu) GetSSHUser() string { return "azureuser" }

func (u *ubuntu) GetImage(arch os.Architecture) (string, error) {
	if arch != os.AMD64Arch {
		return "", fmt.Errorf("%v is not supported", arch)
	}
	return compute.UbuntuLatestURN(), nil
}

func (u *ubuntu) GetDefaultInstanceType(arch os.Architecture) string {
	return os.GetDefaultInstanceType(u.env, arch)
}

type windows struct {
	*os.Windows
	env azure.Environment
}

func newWindows(env azure.Environment) *windows {
	return &windows{
		env:     env,
		Windows: os.NewWindows(),
	}
}

func (*windows) GetSSHUser() string { return "azureuser" }

func (w *windows) GetImage(arch os.Architecture) (string, error) {
	if arch != os.AMD64Arch {
		return "", fmt.Errorf("%v is not supported", arch)
	}
	return compute.WindowsLatestURN(), nil
}

func (w *windows) GetDefaultInstanceType(arch os.Architecture) string {
	return os.GetDefaultInstanceType(&w.env, arch)
}
