package os

import (
	"errors"

	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/aws/ec2/ec2"
	commonos "github.com/DataDog/test-infra-definitions/common/os"
)

type windows struct {
	*commonos.Windows
	env aws.Environment
}

func newWindows(env aws.Environment) *windows {
	return &windows{
		Windows: commonos.NewWindows(&env),
		env:     env,
	}
}

func (*windows) GetSSHUser() string { panic("Not Yet supported") }

func (w *windows) GetImage(arch commonos.Architecture) (string, error) {
	if arch == commonos.ARM64Arch {
		return "", errors.New("ARM64 is not supported for Windows")
	}
	return ec2.GetLatestAMI(w.env, arch,
		"/aws/service/ami-windows-latest/TPM-Windows_Server-2022-English-Full-Base",
		"")
}

func (*windows) GetAMIArch(arch commonos.Architecture) string { return string(arch) }

func (*windows) GetTenancy() string { return "default" }
