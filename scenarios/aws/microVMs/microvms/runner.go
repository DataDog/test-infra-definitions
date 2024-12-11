package microvms

import (
	"errors"

	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/pulumi/pulumi-command/sdk/go/command/local"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Runner struct {
	remoteRunner *command.RemoteRunner
	localRunner  *command.LocalRunner
}

func NewRunner(options ...func(*Runner)) *Runner {
	runner := Runner{}

	for _, opts := range options {
		opts(&runner)
	}

	return &runner
}

func WithRemoteRunner(runner *command.RemoteRunner) func(*Runner) {
	return func(a *Runner) {
		a.remoteRunner = runner
	}
}

func WithLocalRunner(runner *command.LocalRunner) func(*Runner) {
	return func(a *Runner) {
		a.localRunner = runner
	}
}

func (a *Runner) Command(name string, args *command.Args, opts ...pulumi.ResourceOption) (pulumi.Resource, error) {
	if a.remoteRunner != nil {
		return a.remoteRunner.Command(name, args, opts...)
	}
	if a.localRunner != nil {
		return a.localRunner.Command(name, args, opts...)
	}

	panic("no runner initialized")
}

func (a *Runner) RemoteCommand(name string, args *command.Args, opts ...pulumi.ResourceOption) (*remote.Command, error) {
	if a.remoteRunner == nil {
		return nil, errors.New("remote runner not initialized for Runner instance")
	}
	return a.remoteRunner.Command(name, args, opts...)
}

func (a *Runner) LocalCommand(name string, args *command.Args, opts ...pulumi.ResourceOption) (*local.Command, error) {
	if a.localRunner == nil {
		return nil, errors.New("local runner not initialized for Runner instance")
	}
	return a.localRunner.Command(name, args, opts...)
}

func (a *Runner) GetRemoteRunner() (*command.RemoteRunner, error) {
	if a.remoteRunner == nil {
		return nil, errors.New("remote runner not initialized")
	}

	return a.remoteRunner, nil
}

func (a *Runner) GetLocalRunner() (*command.LocalRunner, error) {
	if a.localRunner == nil {
		return nil, errors.New("local runner not initialized")
	}
	return a.localRunner, nil
}
