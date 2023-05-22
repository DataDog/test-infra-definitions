package os

import (
	"errors"

	commonos "github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/resources/aws/ec2"
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

func (*windows) GetSSHUser() string { return "Administrator" }

func (w *windows) GetImage(arch commonos.Architecture) (string, error) {
	if arch == commonos.ARM64Arch {
		return "", errors.New("ARM64 is not supported for Windows")
	}
	return ec2.GetLatestAMI(w.env, arch,
		"/aws/service/ami-windows-latest/Windows_Server-2022-English-Full-Base",
		"")
}

func (*windows) GetAMIArch(arch commonos.Architecture) string { return string(arch) }

func (*windows) GetTenancy() string { return "default" }
