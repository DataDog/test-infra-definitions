package os

import (
	"errors"

	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/common/os"
	oscommon "github.com/DataDog/test-infra-definitions/common/os"
)

type windows struct {
	*oscommon.Windows
	env aws.Environment
}

func newWindows(env aws.Environment) *windows {
	return &windows{
		Windows: oscommon.NewWindows(&env),
		env:     env,
	}
}

func (*windows) GetSSHUser() string { panic("Not Yet supported") }

func (w *windows) GetImage(arch oscommon.Architecture) (string, error) {
	if arch == os.ARM64Arch {
		return "", errors.New("ARM64 is not supported for Windows")
	}
	return os.GetLatestAMI(w.env, arch,
		"/aws/service/ami-windows-latest/TPM-Windows_Server-2022-English-Full-Base",
		"")
}

func (*windows) GetAMIArch(arch oscommon.Architecture) string { return string(arch) }

func (*windows) GetTenancy() string { return "default" }
