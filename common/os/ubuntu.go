package os

import (
	"github.com/DataDog/test-infra-definitions/command"
	"github.com/DataDog/test-infra-definitions/common/config"
)

type Ubuntu struct{ Unix }

func NewUbuntu(env config.Environment) *Ubuntu {
	return &Ubuntu{
		Unix: Unix{env: env},
	}
}

func (*Ubuntu) GetServiceManager() *ServiceManager {
	return NewServiceCmdServiceManager()
}

func (*Ubuntu) CreatePackageManager(runner *command.Runner) (command.PackageManager, error) {
	return NewAptManager(runner), nil
}
