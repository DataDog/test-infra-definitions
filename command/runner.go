package command

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/namer"
	"github.com/pulumi/pulumi-command/sdk/go/command"
	pulumiCommand "github.com/pulumi/pulumi-command/sdk/go/command"
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

func (args *Args) toRemoteCommandArgs(config runnerConfiguration, osCommand OSCommand) *remote.CommandArgs {
	return &remote.CommandArgs{
		Connection: config.connection,
		Create:     osCommand.BuildCommandString(args.Create, args.Environment, args.Sudo, config.user),
		Update:     osCommand.BuildCommandString(args.Update, args.Environment, args.Sudo, config.user),
		Delete:     osCommand.BuildCommandString(args.Delete, args.Environment, args.Sudo, config.user),
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
	osCommand   OSCommand
	provider    *command.Provider
}

func WithUser(user string) func(*Runner) {
	return func(r *Runner) {
		r.config.user = user
	}
}

func WithProvider(provider *pulumiCommand.Provider) func(*Runner) {
	return func(r *Runner) {
		r.provider = provider
	}
}

func NewRunner(
	e config.CommonEnvironment,
	connName string,
	conn remote.ConnectionInput,
	readyFunc func(*Runner) (*remote.Command, error),
	osCommand OSCommand,
	options ...func(*Runner)) (*Runner, error) {
	runner := &Runner{
		e:     e,
		namer: namer.NewNamer(e.Ctx, "remote").WithPrefix(connName),
		config: runnerConfiguration{
			connection: conn,
		},
		osCommand: osCommand,
	}

	for _, opt := range options {
		opt(runner)
	}

	// set default provider if none set
	var err error
	if runner.provider == nil {
		runner.provider, err = pulumiCommand.NewProvider(e.Ctx, runner.namer.ResourceName("provider", connName), &pulumiCommand.ProviderArgs{})
		if err != nil {
			return nil, err
		}
	}

	runner.waitCommand, err = readyFunc(runner)
	if err != nil {
		return nil, err
	}

	return runner, nil
}

func (r *Runner) Command(name string, args *Args, opts ...pulumi.ResourceOption) (*remote.Command, error) {
	if r.waitCommand != nil {
		opts = append(opts, pulumi.DependsOn([]pulumi.Resource{r.waitCommand}))
	}
	if args.Sudo && r.config.user != "" {
		r.e.Ctx.Log.Info(fmt.Sprintf("warning: running sudo command on a runner with user %s, discarding user", r.config.user), nil)
	}
	return remote.NewCommand(r.e.Ctx, r.namer.ResourceName("cmd", name), args.toRemoteCommandArgs(r.config, r.osCommand), append(opts, pulumi.Provider(r.provider))...)
}

type LocalRunner struct {
	e         config.CommonEnvironment
	namer     namer.Namer
	config    runnerConfiguration
	osCommand OSCommand
	provider  *pulumiCommand.Provider
}

func WithLocalUser(user string) func(*LocalRunner) {
	return func(l *LocalRunner) {
		l.config.user = user
	}
}

func WithLocalProvider(provider *pulumiCommand.Provider) func(*LocalRunner) {
	return func(l *LocalRunner) {
		l.provider = provider
	}
}

func NewLocalRunner(e config.CommonEnvironment, osCommand OSCommand, options ...func(*LocalRunner)) (*LocalRunner, error) {
	localRunner := &LocalRunner{
		e:         e,
		namer:     namer.NewNamer(e.Ctx, "local"),
		osCommand: osCommand,
	}

	for _, opt := range options {
		opt(localRunner)
	}

	// set default provider if none set
	var err error
	if localRunner.provider == nil {
		localRunner.provider, err = pulumiCommand.NewProvider(e.Ctx, localRunner.namer.ResourceName("provider"), &pulumiCommand.ProviderArgs{})
		if err != nil {
			return nil, err
		}
	}

	return localRunner, nil
}

func (args *Args) toLocalCommandArgs(config runnerConfiguration, osCommand OSCommand) *local.CommandArgs {
	return &local.CommandArgs{
		Create:   osCommand.BuildCommandString(args.Create, args.Environment, args.Sudo, config.user),
		Update:   osCommand.BuildCommandString(args.Update, args.Environment, args.Sudo, config.user),
		Delete:   osCommand.BuildCommandString(args.Delete, args.Environment, args.Sudo, config.user),
		Triggers: args.Triggers,
		Stdin:    args.Stdin,
	}
}

func (r *LocalRunner) Command(name string, args *Args, opts ...pulumi.ResourceOption) (*local.Command, error) {
	return local.NewCommand(r.e.Ctx, r.namer.ResourceName("cmd", name), args.toLocalCommandArgs(r.config, r.osCommand), append(opts, pulumi.Provider(r.provider))...)
}
