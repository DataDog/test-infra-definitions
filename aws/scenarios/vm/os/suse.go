package os

import (
	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/aws/ec2"
	"github.com/DataDog/test-infra-definitions/command"
	commonos "github.com/DataDog/test-infra-definitions/common/os"
)

type suse struct {
	*unix
	*commonos.Unix
	env aws.Environment
}

func newSuse(env aws.Environment) *suse {
	return &suse{
		unix: &unix{},
		env:  env,
		Unix: commonos.NewUnix(&env),
	}
}
func (*suse) GetSSHUser() string { return "ec2-user" }

func (u *suse) GetImage(arch commonos.Architecture) (string, error) {
	return ec2.GetLatestAMI(u.env, arch,
		"/aws/service/suse/sles/15-sp4/x86_64/latest",
		"/aws/service/suse/sles/15-sp4/arm64/latest",
	)
}

func (*suse) GetServiceManager() *commonos.ServiceManager {
	return commonos.NewSystemCtlServiceManager()
}

func (*suse) CreatePackageManager(runner *command.Runner) (command.PackageManager, error) {
	return newZypperManager(runner), nil
}
