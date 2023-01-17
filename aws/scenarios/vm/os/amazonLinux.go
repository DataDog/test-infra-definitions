package os

import (
	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/aws/ec2/ec2"
	"github.com/DataDog/test-infra-definitions/common/os"
)

type amazonLinux struct {
	*unix
	*os.Unix
	env aws.Environment
}

func newAmazonLinux(env aws.Environment) *amazonLinux {
	return &amazonLinux{
		unix: &unix{},
		env:  env,
		Unix: os.NewUnix(&env),
	}
}
func (*amazonLinux) GetSSHUser() string { return "ec2-user" }

func (u *amazonLinux) GetImage(arch os.Architecture) (string, error) {
	return ec2.SearchAMI(u.env, "137112412989", "amzn2-ami-kernel-5.10-hvm-2.0.*", string(arch))
}

func (u *amazonLinux) GetServiceManager() *os.ServiceManager { return os.NewSystemCtlServiceManager() }
