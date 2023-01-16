package os

import "github.com/DataDog/test-infra-definitions/common/config"

type Ubuntu struct{ unix }

func NewUbuntu(env config.Environment) *Ubuntu {
	return &Ubuntu{
		unix: unix{env: env},
	}
}

func (*Ubuntu) GetServiceManager() *ServiceManager {
	return &ServiceManager{restartCmd: []string{"sudo service datadog-agent restart"}}
}

func (*Ubuntu) GetOSType() OSType { return UbuntuOS }
