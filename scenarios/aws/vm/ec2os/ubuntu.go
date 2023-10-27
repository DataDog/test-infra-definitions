package ec2os

import (
	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/resources/aws/ec2"
)

type ubuntu struct {
	os.Ubuntu
	*unix
	env aws.Environment
}

func newUbuntu(env aws.Environment) *ubuntu {
	return &ubuntu{
		Ubuntu: *os.NewUbuntu(),
		unix:   newUnix(&env),
		env:    env,
	}
}
func (*ubuntu) GetSSHUser() string { return "ubuntu" }

func (u *ubuntu) GetImage(arch os.Architecture) (string, error) {
	return ec2.GetLatestAMI(u.env, arch,
		"/aws/service/canonical/ubuntu/server/jammy/stable/current/amd64/hvm/ebs-gp2/ami-id",
		"/aws/service/canonical/ubuntu/server/jammy/stable/current/arm64/hvm/ebs-gp2/ami-id",
	)
}
