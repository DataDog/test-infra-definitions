package os

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type macOSServiceManager struct {
	e      config.CommonEnvironment
	runner *command.Runner
}

func newMacOSServiceManager(e config.CommonEnvironment, runner *command.Runner) ServiceManager {
	return &macOSServiceManager{e: e, runner: runner}
}

func (s *macOSServiceManager) EnsureRestarted(serviceName string, customizer command.Customizer, opts ...pulumi.ResourceOption) (*remote.Command, error) {
	cmdName := s.e.CommonNamer.ResourceName("running", serviceName)
	cmdArgs := command.Args{
		Sudo:   true,
		Create: pulumi.String(fmt.Sprintf("launchctl stop %s && launchctl start %s", serviceName, serviceName)),
	}
	if customizer != nil {
		cmdName, cmdArgs = customizer(cmdName, cmdArgs)
	}

	return s.runner.Command(cmdName, &cmdArgs, opts...)
}
