package os

import (
	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/aws/ec2/ec2"
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
	return ec2.SearchAMI(u.env, "136693071363", "debian-11-*", string(arch))
}

func (*debian) GetServiceManager() *commonos.ServiceManager {
	return commonos.NewServiceCmdServiceManager()
}
