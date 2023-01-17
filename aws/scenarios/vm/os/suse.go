package os

import (
	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/aws/ec2/ec2"
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
	return ec2.SearchAMI(u.env, "013907871322", "suse-sles-*-hvm-ssd-*", string(arch))
}

func (*suse) GetServiceManager() *commonos.ServiceManager {
	return commonos.NewSystemCtlServiceManager()
}
