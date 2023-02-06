package os

import "github.com/DataDog/test-infra-definitions/common/config"

type Windows struct {
	env config.Environment
}

func NewWindows(env config.Environment) *Windows {
	return &Windows{
		env: env,
	}
}

func (w *Windows) GetDefaultInstanceType(arch Architecture) string {
	return getDefaultInstanceType(w.env, arch)
}

func (*Windows) GetServiceManager() *ServiceManager {
	return &ServiceManager{restartCmd: []string{`%ProgramFiles%\Datadog\Datadog Agent\bin\agent.exe restart-service`}}
}

func (*Windows) GetAgentConfigPath() string { return `C:\ProgramData\Datadog\datadog.yaml` }

func (*Windows) GetAgentInstallCmd(version AgentVersion) string {
	panic("No yet implemented")
}

func (*Windows) GetType() Type {
	return WindowsType
}
