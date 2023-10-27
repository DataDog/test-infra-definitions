package os

import (
	"github.com/DataDog/test-infra-definitions/components/command"
)

var _ RawOS = (*Fedora)(nil)

type Fedora struct {
	*Unix
}

func NewFedora() *Fedora {
	return &Fedora{
		Unix: NewUnix(),
	}
}

func (*Fedora) GetServiceManager() *ServiceManager {
	return NewSystemCtlServiceManager()
}

func (*Fedora) CreatePackageManager(runner *command.Runner) (command.PackageManager, error) {
	return NewDnfManager(runner), nil
}
