package os

import "github.com/DataDog/test-infra-definitions/common"

type Windows struct {
	env common.Environment
}

func NewWindows(env common.Environment) *Windows {
	return &Windows{
		env: env,
	}
}

func (w *Windows) GetDefaultInstanceType(arch Architecture) string {
	return getDefaultInstanceType(w.env, arch)
}

func (*Windows) GetServiceManager() *serviceManager {
	return &serviceManager{restartCmd: []string{`%ProgramFiles%\Datadog\Datadog Agent\bin\agent.exe restart-service`}}
}

func (*Windows) GetConfigPath() string { return `C:\ProgramData\Datadog\datadog.yaml` }

func (*Windows) GetOSType() OSType { return WindowsOS }
