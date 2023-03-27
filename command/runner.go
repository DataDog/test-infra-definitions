package command

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/namer"
	"github.com/pulumi/pulumi-command/sdk/go/command/local"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Args struct {
	Create      pulumi.StringInput
	Update      pulumi.StringInput
	Delete      pulumi.StringInput
	Triggers    pulumi.ArrayInput
	Stdin       pulumi.StringPtrInput
	Environment pulumi.StringMap
	Sudo        bool
}

func (args *Args) toRemoteCommandArgs(config runnerConfiguration, osCommand osCommand) *remote.CommandArgs {
	return &remote.CommandArgs{
		Connection: config.connection,
		Create:     osCommand.BuildCommand(args.Create, args.Environment, args.Sudo, config.user),
		Update:     osCommand.BuildCommand(args.Update, args.Environment, args.Sudo, config.user),
		Delete:     osCommand.BuildCommand(args.Delete, args.Environment, args.Sudo, config.user),
		Triggers:   args.Triggers,
		Stdin:      args.Stdin,
	}
}

type runnerConfiguration struct {
	user       string
	connection remote.ConnectionInput
}

type Runner struct {
	e           config.CommonEnvironment
	namer       namer.Namer
	waitCommand *remote.Command
	config      runnerConfiguration
	osCommand   osCommand
}

func WithUser(user string) func(*Runner) {
	return func(r *Runner) {
		r.config.user = user
	}
}

func NewRunner(
	e config.CommonEnvironment,
	connName string,
	conn remote.ConnectionInput,
	readyFunc func(*Runner) (*remote.Command, error),
	isWindows bool,
	options ...func(*Runner)) (*Runner, error) {
	runner := &Runner{
		e:     e,
		namer: namer.NewNamer(e.Ctx, "remote").WithPrefix(connName),
		config: runnerConfiguration{
			connection: conn,
		},
		osCommand: getOSCommand(isWindows),
	}

	for _, opt := range options {
		opt(runner)
	}

	if readyFunc != nil {
		var err error
		runner.waitCommand, err = readyFunc(runner)
		if err != nil {
			return nil, err
		}
	}

	return runner, nil
}

func (r *Runner) Command(name string, args *Args, opts ...pulumi.ResourceOption) (*remote.Command, error) {
	if r.waitCommand != nil {
		opts = append(opts, pulumi.DependsOn([]pulumi.Resource{r.waitCommand}))
	}

	return remote.NewCommand(r.e.Ctx, r.namer.ResourceName("cmd", name), args.toRemoteCommandArgs(r.config, r.osCommand), opts...)
}

func getOSCommand(isWindows bool) osCommand {
	if isWindows {
		return newWindowsOSCommand()
	}
	return newUnixOSCommand()
}

type LocalRunner struct {
	e         config.CommonEnvironment
	namer     namer.Namer
	config    runnerConfiguration
	osCommand osCommand
}

func WithLocalUser(user string) func(*LocalRunner) {
	return func(l *LocalRunner) {
		l.config.user = user
	}
}

func NewLocalRunner(e config.CommonEnvironment, isWindows bool, options ...func(*LocalRunner)) *LocalRunner {
	localRunner := &LocalRunner{
		e:         e,
		namer:     namer.NewNamer(e.Ctx, "local"),
		osCommand: getOSCommand(isWindows),
	}

	for _, opt := range options {
		opt(localRunner)
	}

	return localRunner
}

func (args *Args) toLocalCommandArgs(config runnerConfiguration, osCommand osCommand) *local.CommandArgs {
	return &local.CommandArgs{
		Create:   osCommand.BuildCommand(args.Create, args.Environment, args.Sudo, config.user),
		Update:   osCommand.BuildCommand(args.Update, args.Environment, args.Sudo, config.user),
		Delete:   osCommand.BuildCommand(args.Delete, args.Environment, args.Sudo, config.user),
		Triggers: args.Triggers,
		Stdin:    args.Stdin,
	}
}

func (r *LocalRunner) Command(name string, args *Args, opts ...pulumi.ResourceOption) (*local.Command, error) {
	return local.NewCommand(r.e.Ctx, r.namer.ResourceName("cmd", name), args.toLocalCommandArgs(r.config, r.osCommand), opts...)
}
