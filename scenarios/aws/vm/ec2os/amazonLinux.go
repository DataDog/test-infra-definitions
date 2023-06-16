package ec2os

import (
	"github.com/DataDog/test-infra-definitions/components/command"
	commonos "github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/resources/aws/ec2"
)

type amazonLinux struct {
	*unix
	*commonos.Unix
	env aws.Environment
}

func newAmazonLinux(env aws.Environment) *amazonLinux {
	return &amazonLinux{
		unix: &unix{},
		env:  env,
		Unix: commonos.NewUnix(&env),
	}
}
func (*amazonLinux) GetSSHUser() string { return "ec2-user" }

func (u *amazonLinux) GetImage(arch commonos.Architecture) (string, error) {
	return ec2.GetLatestAMI(u.env, arch,
		"/aws/service/ami-amazon-linux-latest/amzn2-ami-hvm-x86_64-gp2",
		"/aws/service/ami-amazon-linux-latest/amzn2-ami-hvm-arm64-gp2")
}

func (u *amazonLinux) GetServiceManager() *commonos.ServiceManager {
	return commonos.NewSystemCtlServiceManager()
}

func (*amazonLinux) CreatePackageManager(runner *command.Runner) (command.PackageManager, error) {
	return newYumManager(runner), nil
}
