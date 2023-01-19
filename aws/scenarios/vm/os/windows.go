package os

import (
	"fmt"

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

func (*windows) GetSSHUser() string { return "administrator" }

func (w *windows) GetImage(arch commonos.Architecture) (string, error) {
	if arch != commonos.AMD64Arch {
		return "", fmt.Errorf("the architecture %v is not supported on Windows", arch)
	}
	return ec2.SearchAMI(w.env, "801119661308", "Windows_Server-2022-English-Full-Base-*", string(arch))
}

func (*windows) GetAMIArch(arch commonos.Architecture) string { return string(arch) }

func (*windows) GetTenancy() string { return "default" }
