package agent

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/components/datadog/agentparams"
	"github.com/DataDog/test-infra-definitions/components/os"
	remoteComp "github.com/DataDog/test-infra-definitions/components/remote"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// internal interface to be able to provide the different OS-specific commands
type agentOSManager interface {
	getInstallCommand(version agentparams.PackageVersion) (string, error)
	getAgentConfigFolder() string
	restartAgentServices(triggers pulumi.ArrayInput, opts ...pulumi.ResourceOption) (*remote.Command, error)
}

func getOSManager(host *remoteComp.Host) agentOSManager {
	switch host.OS.Descriptor().Family() { // nolint:exhaustive
	case os.LinuxFamily:
		return newLinuxManager(host)
	case os.WindowsFamily:
		return newWindowsManager(host)
	default:
		panic(fmt.Sprintf("unsupported OS: %v", host.OS.Descriptor().Family()))
	}
}
