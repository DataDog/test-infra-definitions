package os

import (
	"strings"

	"github.com/DataDog/test-infra-definitions/components/command"
)

type MacOS struct{}

func NewMacOS() *MacOS {
	return &MacOS{}
}

func (*MacOS) GetServiceManager() *ServiceManager {
	return &ServiceManager{restartCmd: []string{"launchctl stop com.datadoghq.agent", "launchctl start com.datadoghq.agent"}}
}

func (*MacOS) GetAgentConfigFolder() string { return "~/.datadog-agent" }

func (*MacOS) GetAgentInstallCmd(version AgentVersion) (string, error) {
	return getUnixInstallFormatString("install_mac_os.sh", version), nil
}

func (*MacOS) CreatePackageManager(runner *command.Runner) (command.PackageManager, error) {
	return NewBrewManager(runner), nil
}

func (*MacOS) GetRunAgentCmd(parameters string) string {
	return "datadog-agent " + parameters
}

func (*MacOS) CheckIsAbsPath(path string) bool {
	return strings.HasPrefix(path, "/")
}
