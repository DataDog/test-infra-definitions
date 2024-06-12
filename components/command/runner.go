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

type Runner struct {
	e           config.Env
	namer       namer.Namer
	waitCommand *remote.Command
	config      runnerConfiguration
	osCommand   OSCommand
	options     []pulumi.ResourceOption
}

type RunnerArgs struct {
	ParentResource pulumi.Resource
	ConnectionName string
	Connection     remote.ConnectionInput
	ReadyFunc      ReadyFunc
	User           string
	OSCommand      OSCommand
}

func NewRunner(e config.Env, args RunnerArgs) (*Runner, error) {
	runner := &Runner{
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

func (r *Runner) Command(name string, args *Args, opts ...pulumi.ResourceOption) (*remote.Command, error) {
	if args.Sudo && r.config.user != "" {
		r.e.Ctx().Log.Info(fmt.Sprintf("warning: running sudo command on a runner with user %s, discarding user", r.config.user), nil)
	}

	return remote.NewCommand(r.e.Ctx(), r.namer.ResourceName("cmd", name), args.toRemoteCommandArgs(r.config, r.osCommand), utils.MergeOptions(r.options, opts...)...)
}

func (r *Runner) NewCopyFile(localPath, remotePath string, opts ...pulumi.ResourceOption) (*remote.CopyFile, error) {
	return r.osCommand.NewCopyFile(r, localPath, remotePath, opts...)
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

func (r *LocalRunner) Command(name string, args *Args, opts ...pulumi.ResourceOption) (*local.Command, error) {
	opts = utils.MergeOptions[pulumi.ResourceOption](opts, r.e.WithProviders(config.ProviderCommand))
	return local.NewCommand(r.e.Ctx(), r.namer.ResourceName("cmd", name), args.toLocalCommandArgs(r.config, r.osCommand), opts...)
}
