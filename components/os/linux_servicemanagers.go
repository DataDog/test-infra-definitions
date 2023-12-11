package os

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type systemdServiceManager struct {
	e      config.CommonEnvironment
	runner *command.Runner
}

func newSystemdServiceManager(e config.CommonEnvironment, runner *command.Runner) ServiceManager {
	return &systemdServiceManager{e: e, runner: runner}
}

func (s *systemdServiceManager) EnsureRestarted(serviceName string, triggers pulumi.ArrayInput, opts ...pulumi.ResourceOption) (*remote.Command, error) {
	return s.runner.Command(s.e.CommonNamer.ResourceName("running", serviceName), &command.Args{
		Sudo:     true,
		Create:   pulumi.String("systemctl restart " + serviceName),
		Triggers: triggers,
	}, opts...)
}

// Seems that it was wrongly used for Ubuntu only before.
// type sysVServiceManager struct {
// 	e      config.CommonEnvironment
// 	runner *command.Runner
// }

// func newSysVServiceManager(e config.CommonEnvironment, runner *command.Runner) ServiceManager {
// 	return &sysVServiceManager{e: e, runner: runner}
// }

// func (s *sysVServiceManager) EnsureRunning(serviceName string, triggers pulumi.ArrayInput, opts ...pulumi.ResourceOption) (*remote.Command, error) {
// 	return s.runner.Command(s.e.CommonNamer.ResourceName("running", serviceName), &command.Args{
// 		Sudo:     true,
// 		Create:   pulumi.String("service restart " + serviceName),
// 		Triggers: triggers,
// 	}, opts...)
// }
