package os

import "github.com/DataDog/test-infra-definitions/common"

type Ubuntu struct{ unix }

func NewUbuntu(env common.Environment) *Ubuntu {
	return &Ubuntu{
		unix: unix{env: env},
	}
}

func (*Ubuntu) GetServiceManager() *serviceManager {
	return &serviceManager{restartCmd: []string{"sudo service datadog-agent restart"}}
}

func (*Ubuntu) GetOSType() OSType { return UbuntuOS }
