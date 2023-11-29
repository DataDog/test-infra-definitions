package agent

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/components/datadog/agentparams"
	"github.com/DataDog/test-infra-definitions/components/os"
)

// internal interface to be able to provide the different OS-specific commands
type agentOSManager interface {
	getInstallCommand(version agentparams.PackageVersion) (string, error)
	getAgentConfigFolder() string
}

func getOSManager(targetOS os.OS) agentOSManager {
	switch targetOS.Descriptor().Family() { // nolint:exhaustive
	case os.LinuxFamily:
		return newLinuxManager(targetOS)
	case os.WindowsFamily:
		return newWindowsManager(targetOS)
	default:
		panic(fmt.Sprintf("unsupported OS: %v", targetOS.Descriptor().Family()))
	}
}
