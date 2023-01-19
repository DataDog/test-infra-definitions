package os

import (
	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/aws/ec2/ec2"
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

func (*windows) GetSSHUser() string { return "administrator" }

func (w *windows) GetImage(arch oscommon.Architecture) (string, error) {
	return ec2.SearchAMI(w.env, "801119661308", "Windows_Server-2022-English-Full-Base-*", string(arch))
}

func (*windows) GetAMIArch(arch oscommon.Architecture) string { return string(arch) }

func (*windows) GetTenancy() string { return "default" }
