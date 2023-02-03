package os

import (
	"github.com/DataDog/test-infra-definitions/aws"
	commonos "github.com/DataDog/test-infra-definitions/common/os"
)

type ubuntu struct {
	commonos.Ubuntu
	*unix
	env aws.Environment
}

func newUbuntu(env aws.Environment) *ubuntu {
	return &ubuntu{
		Ubuntu: *commonos.NewUbuntu(&env),
		unix:   &unix{},
		env:    env,
	}
}
func (*ubuntu) GetSSHUser() string { return "ubuntu" }

func (u *ubuntu) GetImage(arch commonos.Architecture) (string, error) {
	return commonos.GetLatestAMI(u.env, arch,
		"/aws/service/canonical/ubuntu/server/jammy/stable/current/amd64/hvm/ebs-gp2/ami-id",
		"/aws/service/canonical/ubuntu/server/jammy/stable/current/arm64/hvm/ebs-gp2/ami-id",
	)
}
