package os

import (
	"github.com/DataDog/test-infra-definitions/components/command"
)

var _ RawOS = (*RockyLinux)(nil)

type RockyLinux struct {
	*Unix
}

func NewRockyLinux() *RockyLinux {
	return &RockyLinux{
		Unix: NewUnix(),
	}
}

func (*RockyLinux) GetServiceManager() *ServiceManager {
	return NewSystemCtlServiceManager()
}

func (*RockyLinux) CreatePackageManager(runner *command.Runner) (command.PackageManager, error) {
	return NewYumManager(runner), nil
}
