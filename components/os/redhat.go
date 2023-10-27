package os

import (
	"github.com/DataDog/test-infra-definitions/components/command"
)

var _ RawOS = (*RedHat)(nil)

type RedHat struct {
	*Unix
}

func NewRedHat() *RedHat {
	return &RedHat{
		Unix: NewUnix(),
	}
}

func (*RedHat) GetServiceManager() *ServiceManager {
	return NewSystemCtlServiceManager()
}

func (*RedHat) CreatePackageManager(runner *command.Runner) (command.PackageManager, error) {
	return NewYumManager(runner), nil
}
