package os

import (
	"github.com/DataDog/test-infra-definitions/components/command"
)

type Centos struct {
	*Unix
}

var _ RawOS = (*Centos)(nil)

func NewCentos() *Centos {
	return &Centos{
		Unix: NewUnix(),
	}
}

func (*Centos) GetServiceManager() *ServiceManager {
	return NewSystemCtlServiceManager()
}

func (*Centos) CreatePackageManager(runner *command.Runner) (command.PackageManager, error) {
	return NewYumManager(runner), nil
}
