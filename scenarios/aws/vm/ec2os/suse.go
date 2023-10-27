package ec2os

import (
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/resources/aws/ec2"
)

type suse struct {
	*unix
	*os.Unix
	env aws.Environment
}

func newSuse(env aws.Environment) *suse {
	return &suse{
		unix: newUnix(&env),
		env:  env,
		Unix: os.NewUnix(),
	}
}
func (*suse) GetSSHUser() string { return "ec2-user" }

func (u *suse) GetImage(arch os.Architecture) (string, error) {
	return ec2.GetLatestAMI(u.env, arch,
		"/aws/service/suse/sles/15-sp4/x86_64/latest",
		"/aws/service/suse/sles/15-sp4/arm64/latest",
	)
}

func (*suse) GetServiceManager() *os.ServiceManager {
	return os.NewSystemCtlServiceManager()
}

func (*suse) CreatePackageManager(runner *command.Runner) (command.PackageManager, error) {
	return newZypperManager(runner), nil
}
