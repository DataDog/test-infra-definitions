package os

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type windowsServiceManager struct {
	e      config.Env
	runner *command.RemoteRunner
}

func newWindowsServiceManager(e config.Env, runner *command.RemoteRunner) ServiceManager {
	return &windowsServiceManager{e: e, runner: runner}
}

func (s *windowsServiceManager) EnsureRestarted(serviceName string, transform command.Transformer, opts ...pulumi.ResourceOption) (*remote.Command, error) {
	cmdName := s.e.CommonNamer().ResourceName("running", serviceName)
	cmdArgs := command.Args{
		Create: pulumi.String("Restart-Service -Name " + serviceName),
	}

	// If a transform is provided, use it to modify the command name and args
	if transform != nil {
		cmdName, cmdArgs = transform(cmdName, cmdArgs)
	}

	return s.runner.Command(cmdName, &cmdArgs, opts...)
}
