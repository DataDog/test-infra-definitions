package os

import (
	"github.com/DataDog/test-infra-definitions/components/command"
)

var _ RawOS = (*Debian)(nil)

type Debian struct {
	*Unix
}

func NewDebian() *Debian {
	return &Debian{
		Unix: NewUnix(),
	}
}

func (*Debian) GetServiceManager() *ServiceManager {
	return NewServiceCmdServiceManager()
}

func (*Debian) CreatePackageManager(runner *command.Runner) (command.PackageManager, error) {
	return NewAptManager(runner), nil
}
