package os

type MacOS struct{}

func NewMacOS() *MacOS {
	return &MacOS{}
}

func (*MacOS) GetServiceManager() *ServiceManager {
	return &ServiceManager{restartCmd: []string{"launchctl stop com.datadoghq.agent", "launchctl start com.datadoghq.agent"}}
}

func (*MacOS) GetAgentConfigPath() string { return "~/.datadog-agent/datadog.yaml" }

func (*MacOS) GetAgentInstallCmd(version AgentVersion) string {
	return getUnixInstallFormatString("install_mac_os.sh", version)
}
