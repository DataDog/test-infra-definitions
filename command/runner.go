package command

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/namer"
	"github.com/DataDog/test-infra-definitions/common/utils"
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

func (args *Args) toLocalCommandArgs(config runnerConfiguration, osCommand OSCommand) *local.CommandArgs {
	return &local.CommandArgs{
		Create:   osCommand.BuildCommandString(args.Create, args.Environment, args.Sudo, config.user),
		Update:   osCommand.BuildCommandString(args.Update, args.Environment, args.Sudo, config.user),
		Delete:   osCommand.BuildCommandString(args.Delete, args.Environment, args.Sudo, config.user),
		Triggers: args.Triggers,
		Stdin:    args.Stdin,
	}
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
	options     []pulumi.ResourceOption
}

type RunnerArgs struct {
	ParentResource pulumi.Resource
	ConnectionName string
	Connection     remote.ConnectionInput
	ReadyFunc      func(*Runner) (*remote.Command, error)
	User           string
	OSCommand      OSCommand
}

func NewRunner(e config.CommonEnvironment, args RunnerArgs) (*Runner, error) {
	runner := &Runner{
		e:     e,
		namer: namer.NewNamer(e.Ctx, "remote").WithPrefix(args.ConnectionName),
		config: runnerConfiguration{
			connection: args.Connection,
			user:       args.User,
		},
		osCommand: args.OSCommand,
	}

	if args.ParentResource != nil {
		runner.options = append(runner.options, pulumi.Parent(args.ParentResource), pulumi.DeletedWith(args.ParentResource))
	}

	var err error
	runner.waitCommand, err = args.ReadyFunc(runner)
	if err != nil {
		return nil, err
	}
	runner.options = append(runner.options, utils.PulumiDependsOn(runner.waitCommand))

	return runner, nil
}

func (r *Runner) Command(name string, args *Args, opts ...pulumi.ResourceOption) (*remote.Command, error) {
	opts = append(opts, r.options...)
	if args.Sudo && r.config.user != "" {
		r.e.Ctx.Log.Info(fmt.Sprintf("warning: running sudo command on a runner with user %s, discarding user", r.config.user), nil)
	}
	depends := append(opts, pulumi.Provider(r.e.CommandProvider))
	return remote.NewCommand(r.e.Ctx, r.namer.ResourceName("cmd", name), args.toRemoteCommandArgs(r.config, r.osCommand), depends...)
}

type LocalRunner struct {
	e         config.CommonEnvironment
	namer     namer.Namer
	config    runnerConfiguration
	osCommand OSCommand
}

type LocalRunnerArgs struct {
	User      string
	OSCommand OSCommand
}

func NewLocalRunner(e config.CommonEnvironment, args LocalRunnerArgs) *LocalRunner {
	localRunner := &LocalRunner{
		e:         e,
		namer:     namer.NewNamer(e.Ctx, "local"),
		osCommand: args.OSCommand,
		config: runnerConfiguration{
			user: args.User,
		},
	}

	return localRunner
}

func (r *LocalRunner) Command(name string, args *Args, opts ...pulumi.ResourceOption) (*local.Command, error) {
	depends := append(opts, pulumi.Provider(r.e.CommandProvider))
	return local.NewCommand(r.e.Ctx, r.namer.ResourceName("cmd", name), args.toLocalCommandArgs(r.config, r.osCommand), depends...)
}
