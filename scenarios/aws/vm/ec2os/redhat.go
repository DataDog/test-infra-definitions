package ec2os

import (
	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/resources/aws/ec2"
)

type redHat struct {
	*unix
	*os.RedHat
	env aws.Environment
}

func newRedHat(env aws.Environment) *redHat {
	return &redHat{
		unix:   newUnix(&env),
		env:    env,
		RedHat: os.NewRedHat(),
	}
}
func (*redHat) GetSSHUser() string { return "ec2-user" }

func (u *redHat) GetImage(arch os.Architecture) (string, error) {
	return ec2.SearchAMI(u.env, "309956199498", "RHEL-9.1.0_HVM-*-2-Hourly2-GP2", string(arch))
}
