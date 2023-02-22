package os

import (
	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/aws/ec2/ec2"
	"github.com/DataDog/test-infra-definitions/command"
	commonos "github.com/DataDog/test-infra-definitions/common/os"
)

type debian struct {
	*unix
	*commonos.Unix
	env aws.Environment
}

func newDebian(env aws.Environment) *debian {
	return &debian{
		unix: &unix{},
		env:  env,
		Unix: commonos.NewUnix(&env),
	}
}
func (*debian) GetSSHUser() string { return "admin" }

func (u *debian) GetImage(arch commonos.Architecture) (string, error) {
	return ec2.GetLatestAMI(u.env, arch,
		"/aws/service/debian/release/bullseye/latest/amd64",
		"/aws/service/debian/release/bullseye/latest/arm64")
}

func (*debian) GetServiceManager() *commonos.ServiceManager {
	return commonos.NewServiceCmdServiceManager()
}

func (*debian) CreatePackageManager(runner *command.Runner) (command.PackageManager, error) {
	return commonos.NewAptManager(runner), nil
}
