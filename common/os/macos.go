package os

import "github.com/DataDog/test-infra-definitions/command"

type MacOS struct{}

func NewMacOS() *MacOS {
	return &MacOS{}
}

func (*MacOS) GetServiceManager() *ServiceManager {
	return &ServiceManager{restartCmd: []string{"launchctl stop com.datadoghq.agent", "launchctl start com.datadoghq.agent"}}
}

func (*MacOS) GetAgentConfigPath() string { return "~/.datadog-agent/datadog.yaml" }

func (*MacOS) GetAgentInstallCmd(version AgentVersion) (string, error) {
	return getUnixInstallFormatString("install_mac_os.sh", version), nil
}

func (*MacOS) CreatePackageManager(runner *command.Runner) (command.PackageManager, error) {
	return NewBrewManager(runner), nil
}
