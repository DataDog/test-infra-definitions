package os

import (
	"github.com/DataDog/test-infra-definitions/components/command"
)

var _ RawOS = (*Ubuntu)(nil)

type Ubuntu struct{ *Unix }

func NewUbuntu() *Ubuntu {
	return &Ubuntu{
		Unix: NewUnix(),
	}
}

func (*Ubuntu) GetServiceManager() *ServiceManager {
	return NewServiceCmdServiceManager()
}

func (*Ubuntu) CreatePackageManager(runner *command.Runner) (command.PackageManager, error) {
	return NewAptManager(runner), nil
}
