package ec2os

import (
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/resources/aws/ec2"
)

type debian struct {
	*unix
	*os.Unix
	env aws.Environment
}

func newDebian(env aws.Environment) *debian {
	return &debian{
		unix: newUnix(&env),
		env:  env,
		Unix: os.NewUnix(),
	}
}
func (*debian) GetSSHUser() string { return "admin" }

func (u *debian) GetImage(arch os.Architecture) (string, error) {
	return ec2.GetLatestAMI(u.env, arch,
		"/aws/service/debian/release/bullseye/latest/amd64",
		"/aws/service/debian/release/bullseye/latest/arm64")
}

func (*debian) GetServiceManager() *os.ServiceManager {
	return os.NewServiceCmdServiceManager()
}

func (*debian) CreatePackageManager(runner *command.Runner) (command.PackageManager, error) {
	return os.NewAptManager(runner), nil
}
