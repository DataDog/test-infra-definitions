package ec2os

import (
	"github.com/DataDog/test-infra-definitions/components/command"
	commonos "github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/resources/aws/ec2"
)

type redHat struct {
	*unix
	*commonos.Unix
	env aws.Environment
}

func newRedHat(env aws.Environment) *redHat {
	return &redHat{
		unix: &unix{},
		env:  env,
		Unix: commonos.NewUnix(&env),
	}
}
func (*redHat) GetSSHUser() string { return "ec2-user" }

func (u *redHat) GetImage(arch commonos.Architecture) (string, error) {
	return ec2.SearchAMI(u.env, "309956199498", "RHEL-9.1.0_HVM-*-2-Hourly2-GP2", string(arch))
}

func (*redHat) GetServiceManager() *commonos.ServiceManager {
	return commonos.NewSystemCtlServiceManager()
}

func (*redHat) CreatePackageManager(runner *command.Runner) (command.PackageManager, error) {
	return newYumManager(runner), nil
}
