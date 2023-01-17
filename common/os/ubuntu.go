package os

import "github.com/DataDog/test-infra-definitions/common/config"

type Ubuntu struct{ Unix }

func NewUbuntu(env config.Environment) *Ubuntu {
	return &Ubuntu{
		Unix: Unix{env: env},
	}
}

func (*Ubuntu) GetServiceManager() *ServiceManager {
	return &ServiceManager{restartCmd: []string{"sudo service datadog-agent restart"}}
}

func (*Ubuntu) GetType() Type {
	return UbuntuType
}
