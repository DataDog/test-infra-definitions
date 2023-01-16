package os

type MacOS struct{}

func NewMacOS() *MacOS {
	return &MacOS{}
}

func (*MacOS) GetServiceManager() *ServiceManager {
	return &ServiceManager{restartCmd: []string{"launchctl stop com.datadoghq.agent", "launchctl start com.datadoghq.agent"}}
}

func (*MacOS) GetAgentConfigPath() string { return "~/.datadog-agent/datadog.yaml" }

func (*MacOS) GetOSType() OSType { return MacosOS }
