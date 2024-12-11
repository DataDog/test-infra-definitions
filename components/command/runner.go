package command

import (
	"fmt"

	"github.com/pulumi/pulumi-command/sdk/go/command/local"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/namer"
	"github.com/DataDog/test-infra-definitions/common/utils"
)

type Args struct {
	Create                   pulumi.StringInput
	Update                   pulumi.StringInput
	Delete                   pulumi.StringInput
	Triggers                 pulumi.ArrayInput
	Stdin                    pulumi.StringPtrInput
	Environment              pulumi.StringMap
	RequirePasswordFromStdin bool
	Sudo                     bool
}

func (args *Args) toLocalCommandArgs(config runnerConfiguration, osCommand OSCommand) *local.CommandArgs {
	return &local.CommandArgs{
		Create:   osCommand.BuildCommandString(args.Create, args.Environment, args.Sudo, args.RequirePasswordFromStdin, config.user),
		Update:   osCommand.BuildCommandString(args.Update, args.Environment, args.Sudo, args.RequirePasswordFromStdin, config.user),
		Delete:   osCommand.BuildCommandString(args.Delete, args.Environment, args.Sudo, args.RequirePasswordFromStdin, config.user),
		Triggers: args.Triggers,
		Stdin:    args.Stdin,
	}
}

func (args *Args) toRemoteCommandArgs(config runnerConfiguration, osCommand OSCommand) *remote.CommandArgs {
	return &remote.CommandArgs{
		Connection: config.connection,
		Create:     osCommand.BuildCommandString(args.Create, args.Environment, args.Sudo, args.RequirePasswordFromStdin, config.user),
		Update:     osCommand.BuildCommandString(args.Update, args.Environment, args.Sudo, args.RequirePasswordFromStdin, config.user),
		Delete:     osCommand.BuildCommandString(args.Delete, args.Environment, args.Sudo, args.RequirePasswordFromStdin, config.user),
		Triggers:   args.Triggers,
		Stdin:      args.Stdin,
	}
}

// Transformer is a function that can be used to modify the command name and args.
// Examples: swapping `args.Delete` with `args.Create`, or adding `args.Triggers`, or editing the name
type Transformer func(name string, args Args) (string, Args)

type runnerConfiguration struct {
	user       string
	connection remote.ConnectionInput
}

type Command interface {
	pulumi.Resource

	StdoutOutput() pulumi.StringOutput
	StderrOutput() pulumi.StringOutput
}

type LocalCommand struct {
	local.Command
}

type RemoteCommand struct {
	remote.Command
}

var _ Command = &RemoteCommand{}
var _ Command = &LocalCommand{}

func (c *LocalCommand) StdoutOutput() pulumi.StringOutput {
	return c.Command.Stdout
}

func (c *LocalCommand) StderrOutput() pulumi.StringOutput {
	return c.Command.Stderr
}

func (c *RemoteCommand) StdoutOutput() pulumi.StringOutput {
	return c.Command.Stdout
}

func (c *RemoteCommand) StderrOutput() pulumi.StringOutput {
	return c.Command.Stderr
}

type Runner interface {
	Environment() config.Env
	Namer() namer.Namer
	Config() runnerConfiguration
	OsCommand() OSCommand

	Command(name string, args *Args, opts ...pulumi.ResourceOption) (Command, error)
	NewCopyFile(name string, localPath, remotePath pulumi.StringInput, opts ...pulumi.ResourceOption) (pulumi.Resource, error)
	PulumiOptions() []pulumi.ResourceOption
}

var _ Runner = &RemoteRunner{}
var _ Runner = &LocalRunner{}

type RemoteRunner struct {
	e           config.Env
	namer       namer.Namer
	waitCommand Command
	config      runnerConfiguration
	osCommand   OSCommand
	options     []pulumi.ResourceOption
}

type RemoteRunnerArgs struct {
	ParentResource pulumi.Resource
	ConnectionName string
	Connection     remote.ConnectionInput
	ReadyFunc      ReadyFunc
	User           string
	OSCommand      OSCommand
}

func NewRemoteRunner(e config.Env, args RemoteRunnerArgs) (*RemoteRunner, error) {
	runner := &RemoteRunner{
		e:     e,
		namer: namer.NewNamer(e.Ctx(), "remote").WithPrefix(args.ConnectionName),
		config: runnerConfiguration{
			connection: args.Connection,
			user:       args.User,
		},
		osCommand: args.OSCommand,
		options: []pulumi.ResourceOption{
			e.WithProviders(config.ProviderCommand),
		},
	}

	if args.ParentResource != nil {
		runner.options = append(runner.options, pulumi.Parent(args.ParentResource), pulumi.DeletedWith(args.ParentResource))
	}

	if args.ReadyFunc != nil {
		var err error
		runner.waitCommand, err = args.ReadyFunc(runner)
		if err != nil {
			return nil, err
		}
		runner.options = append(runner.options, utils.PulumiDependsOn(runner.waitCommand))
	}

	return runner, nil
}

func (r *RemoteRunner) Environment() config.Env {
	return r.e
}

func (r *RemoteRunner) Namer() namer.Namer {
	return r.namer
}

func (r *RemoteRunner) Config() runnerConfiguration {
	return r.config
}

func (r *RemoteRunner) OsCommand() OSCommand {
	return r.osCommand
}

func (r *RemoteRunner) Command(name string, args *Args, opts ...pulumi.ResourceOption) (Command, error) {
	if args.Sudo && r.config.user != "" {
		r.e.Ctx().Log.Info(fmt.Sprintf("warning: running sudo command on a runner with user %s, discarding user", r.config.user), nil)
	}

	cmd, err := remote.NewCommand(r.e.Ctx(), r.namer.ResourceName("cmd", name), args.toRemoteCommandArgs(r.config, r.osCommand), utils.MergeOptions(r.options, opts...)...)

	return &RemoteCommand{*cmd}, err
}

func (r *RemoteRunner) NewCopyFile(name string, localPath, remotePath pulumi.StringInput, opts ...pulumi.ResourceOption) (pulumi.Resource, error) {
	return r.osCommand.NewCopyFile(r, name, localPath, remotePath, opts...)
}

func (r *RemoteRunner) PulumiOptions() []pulumi.ResourceOption {
	return r.options
}

type LocalRunner struct {
	e         config.Env
	namer     namer.Namer
	config    runnerConfiguration
	osCommand OSCommand
}

type LocalRunnerArgs struct {
	User      string
	OSCommand OSCommand
}

func NewLocalRunner(e config.Env, args LocalRunnerArgs) *LocalRunner {
	localRunner := &LocalRunner{
		e:         e,
		namer:     namer.NewNamer(e.Ctx(), "local"),
		osCommand: args.OSCommand,
		config: runnerConfiguration{
			user: args.User,
		},
	}

	return localRunner
}

func (r *LocalRunner) Environment() config.Env {
	return r.e
}

func (r *LocalRunner) Namer() namer.Namer {
	return r.namer
}

func (r *LocalRunner) Config() runnerConfiguration {
	return r.config
}

func (r *LocalRunner) OsCommand() OSCommand {
	return r.osCommand
}

func (r *LocalRunner) Command(name string, args *Args, opts ...pulumi.ResourceOption) (Command, error) {
	opts = utils.MergeOptions[pulumi.ResourceOption](opts, r.e.WithProviders(config.ProviderCommand))
	cmd, err := local.NewCommand(r.e.Ctx(), r.namer.ResourceName("cmd", name), args.toLocalCommandArgs(r.config, r.osCommand), opts...)

	return &LocalCommand{*cmd}, err
}

func (r *LocalRunner) NewCopyFile(name string, localPath, remotePath pulumi.StringInput, opts ...pulumi.ResourceOption) (pulumi.Resource, error) {
	return r.osCommand.NewCopyFile(r, name, localPath, remotePath, opts...)
}

func (r *LocalRunner) PulumiOptions() []pulumi.ResourceOption {
	return []pulumi.ResourceOption{}
}
