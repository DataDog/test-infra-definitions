package os

import (
	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/common/os"
)

type ubuntu struct {
	os.Ubuntu
	*unix
	env aws.Environment
}

func newUbuntu(env aws.Environment) *ubuntu {
	return &ubuntu{
		Ubuntu: *os.NewUbuntu(&env),
		unix:   &unix{},
		env:    env,
	}
}
func (*ubuntu) GetSSHUser() string { return "ubuntu" }

func (u *ubuntu) GetImage(arch os.Architecture) (string, error) {
	return os.GetLatestAMI(u.env, arch,
		"/aws/service/canonical/ubuntu/server/jammy/stable/current/amd64/hvm/ebs-gp2/ami-id",
		"/aws/service/canonical/ubuntu/server/jammy/stable/current/arm64/hvm/ebs-gp2/ami-id",
	)
}
