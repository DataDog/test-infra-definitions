package os

import (
	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/aws/ec2/ec2"
	"github.com/DataDog/test-infra-definitions/common/os"
	oscommon "github.com/DataDog/test-infra-definitions/common/os"
)

type windows struct {
	*oscommon.Windows
	env aws.Environment
}

func newWindows(env aws.Environment) *windows {
	return &windows{
		Windows: os.NewWindows(&env),
		env:     env,
	}
}

func (*windows) GetSSHUser() string { panic("Not Yet supported") }

func (w *windows) GetImage(arch os.Architecture) (string, error) {
	return ec2.SearchAMI(w.env, "801119661308", "Windows_Server-2022-English-Full-Base-*", string(arch))
}

func (*windows) GetAMIArch(arch os.Architecture) string { return string(arch) }

func (*windows) GetTenancy() string { return "default" }
