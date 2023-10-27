package os

import (
	"github.com/DataDog/test-infra-definitions/components/command"
)

var _ RawOS = (*AmazonLinux)(nil)

type AmazonLinux struct {
	*Unix
}

func NewAmazonLinux() *AmazonLinux {
	return &AmazonLinux{
		Unix: NewUnix(),
	}
}
func (*AmazonLinux) GetSSHUser() string { return "ec2-user" }

func (u *AmazonLinux) GetServiceManager() *ServiceManager {
	return NewSystemCtlServiceManager()
}

func (*AmazonLinux) CreatePackageManager(runner *command.Runner) (command.PackageManager, error) {
	return NewYumManager(runner), nil
}
