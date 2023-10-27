package ec2os

import (
	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/resources/aws/ec2"
)

type debian struct {
	*unix
	*os.Debian
	env aws.Environment
}

func newDebian(env aws.Environment) *debian {
	return &debian{
		unix:   newUnix(&env),
		env:    env,
		Debian: os.NewDebian(),
	}
}
func (*debian) GetSSHUser() string { return "admin" }

func (u *debian) GetImage(arch os.Architecture) (string, error) {
	return ec2.GetLatestAMI(u.env, arch,
		"/aws/service/debian/release/bullseye/latest/amd64",
		"/aws/service/debian/release/bullseye/latest/arm64")
}
