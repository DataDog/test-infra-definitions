package ec2os

import (
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/resources/aws/ec2"
)

type redHat struct {
	*unix
	*os.Unix
	env aws.Environment
}

func newRedHat(env aws.Environment) *redHat {
	return &redHat{
		unix: newUnix(&env),
		env:  env,
		Unix: os.NewUnix(),
	}
}
func (*redHat) GetSSHUser() string { return "ec2-user" }

func (u *redHat) GetImage(arch os.Architecture) (string, error) {
	return ec2.SearchAMI(u.env, "309956199498", "RHEL-9.1.0_HVM-*-2-Hourly2-GP2", string(arch))
}

func (*redHat) GetServiceManager() *os.ServiceManager {
	return os.NewSystemCtlServiceManager()
}

func (*redHat) CreatePackageManager(runner *command.Runner) (command.PackageManager, error) {
	return newYumManager(runner), nil
}
