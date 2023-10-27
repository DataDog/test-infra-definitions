package ec2os

import (
	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/resources/aws/ec2"
)

type amazonLinux struct {
	*unix
	*os.AmazonLinux
	env aws.Environment
}

func newAmazonLinux(env aws.Environment) *amazonLinux {
	return &amazonLinux{
		unix:        newUnix(&env),
		env:         env,
		AmazonLinux: os.NewAmazonLinux(),
	}
}
func (*amazonLinux) GetSSHUser() string { return "ec2-user" }

func (u *amazonLinux) GetImage(arch os.Architecture) (string, error) {
	return ec2.GetLatestAMI(u.env, arch,
		"/aws/service/ami-amazon-linux-latest/amzn2-ami-hvm-x86_64-gp2",
		"/aws/service/ami-amazon-linux-latest/amzn2-ami-hvm-arm64-gp2")
}
