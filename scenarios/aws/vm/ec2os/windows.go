package ec2os

import (
	"errors"

	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/resources/aws/ec2"
)

type windows struct {
	*os.Windows
	env aws.Environment
}

func newWindows(env aws.Environment) *windows {
	return &windows{
		Windows: os.NewWindows(),
		env:     env,
	}
}

func (*windows) GetSSHUser() string { return "Administrator" }

func (w *windows) GetImage(arch os.Architecture) (string, error) {
	if arch == os.ARM64Arch {
		return "", errors.New("ARM64 is not supported for Windows")
	}
	return ec2.GetLatestAMI(w.env, arch,
		"/aws/service/ami-windows-latest/Windows_Server-2022-English-Full-Base",
		"")
}

func (*windows) GetAMIArch(arch os.Architecture) string { return string(arch) }

func (*windows) GetTenancy() string { return "default" }

func (w *windows) GetDefaultInstanceType(arch os.Architecture) string {
	return os.GetDefaultInstanceType(&w.env, arch)
}
