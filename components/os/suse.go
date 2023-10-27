package os

import (
	"github.com/DataDog/test-infra-definitions/components/command"
)

var _ RawOS = (*Suse)(nil)

type Suse struct {
	*Unix
}

func NewSuse() *Suse {
	return &Suse{
		Unix: NewUnix(),
	}
}

func (*Suse) GetServiceManager() *ServiceManager {
	return NewSystemCtlServiceManager()
}

func (*Suse) CreatePackageManager(runner *command.Runner) (command.PackageManager, error) {
	return NewZypperManager(runner), nil
}
