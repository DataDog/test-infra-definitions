package os

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type windowsServiceManager struct {
	e      config.CommonEnvironment
	runner *command.Runner
}

func newWindowsServiceManager(e config.CommonEnvironment, runner *command.Runner) ServiceManager {
	return &windowsServiceManager{e: e, runner: runner}
}

func (s *windowsServiceManager) EnsureRestarted(serviceName string, customizer command.Customizer, opts ...pulumi.ResourceOption) (*remote.Command, error) {
	cmdName := s.e.CommonNamer.ResourceName("running", serviceName)
	cmdArgs := command.Args{
		Create: pulumi.String("Restart-Service -Name " + serviceName),
	}
	if customizer != nil {
		cmdName, cmdArgs = customizer(cmdName, cmdArgs)
	}

	return s.runner.Command(cmdName, &cmdArgs, opts...)
}
