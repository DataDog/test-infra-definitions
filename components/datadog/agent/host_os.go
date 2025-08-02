package agent

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/components/datadog/agentparams"

	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/components/os"
	remoteComp "github.com/DataDog/test-infra-definitions/components/remote"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// internal interface to be able to provide the different OS-specific commands
type agentOSManager interface {
	directInstallCommand(env config.Env, packagePath string, version agentparams.PackageVersion, additionalInstallParameters []string, opts ...pulumi.ResourceOption) (command.Command, error)
	getInstallCommand(version agentparams.PackageVersion, additionalInstallParameters []string) (string, error)
	getAgentConfigFolder() string
	restartAgentServices(transform command.Transformer, opts ...pulumi.ResourceOption) (command.Command, error)
	ensureAgentUninstalled(version agentparams.PackageVersion, opts ...pulumi.ResourceOption) (command.Command, error)
}

func getOSManager(host *remoteComp.Host) agentOSManager {
	switch host.OS.Descriptor().Family() {
	case os.LinuxFamily:
		return newLinuxManager(host)
	case os.WindowsFamily:
		return newWindowsManager(host)
	case os.MacOSFamily, os.UnknownFamily:
		fallthrough
	default:
		panic(fmt.Sprintf("unsupported OS: %v", host.OS.Descriptor().Family()))
	}
}
