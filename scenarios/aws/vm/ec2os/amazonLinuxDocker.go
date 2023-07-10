package ec2os

import (
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/resources/aws/ec2"
)

type amazonLinuxDocker struct {
	*unix
	*os.Unix
	env aws.Environment
}

func newAmazonLinuxDocker(env aws.Environment) *amazonLinuxDocker {
	return &amazonLinuxDocker{
		unix: &unix{},
		env:  env,
		Unix: os.NewUnix(&env),
	}
}
func (*amazonLinuxDocker) GetSSHUser() string { return "ec2-user" }

func (u *amazonLinuxDocker) GetImage(arch os.Architecture) (string, error) {
	return ec2.GetLatestAMI(u.env, arch,
		"/aws/service/ecs/optimized-ami/amazon-linux-2/recommended/image_id",
		"/aws/service/ecs/optimized-ami/amazon-linux-2/arm64/recommended/image_id")
}

func (u *amazonLinuxDocker) GetServiceManager() *os.ServiceManager {
	return os.NewSystemCtlServiceManager()
}

func (*amazonLinuxDocker) CreatePackageManager(runner *command.Runner) (command.PackageManager, error) {
	return newYumManager(runner), nil
}
