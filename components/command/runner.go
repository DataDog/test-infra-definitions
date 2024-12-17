package command

import (
	"fmt"
	"path/filepath"

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
	// Only used for local commands
	LocalAssetPaths pulumi.StringArrayInput
	LocalDir        pulumi.StringInput
}

func (args *Args) toLocalCommandArgs(config RunnerConfiguration, osCommand OSCommand) (*local.CommandArgs, error) {
	return &local.CommandArgs{
		Create:      osCommand.BuildCommandString(args.Create, args.Environment, args.Sudo, args.RequirePasswordFromStdin, config.user),
		Update:      osCommand.BuildCommandString(args.Update, args.Environment, args.Sudo, args.RequirePasswordFromStdin, config.user),
		Delete:      osCommand.BuildCommandString(args.Delete, args.Environment, args.Sudo, args.RequirePasswordFromStdin, config.user),
		Environment: args.Environment,
		Triggers:    args.Triggers,
		Stdin:       args.Stdin,
		AssetPaths:  args.LocalAssetPaths,
		Dir:         args.LocalDir,
	}, nil
}

func (args *Args) toRemoteCommandArgs(config RunnerConfiguration, osCommand OSCommand) (*remote.CommandArgs, error) {
	// Ensure no local arguments are passed to remote commands
	if args.LocalAssetPaths != nil {
		return nil, fmt.Errorf("local asset paths are not supported in remote commands")
	}
	if args.LocalDir != nil {
		return nil, fmt.Errorf("local dir is not supported in remote commands")
	}

	return &remote.CommandArgs{
		Connection:  config.connection,
		Create:      osCommand.BuildCommandString(args.Create, args.Environment, args.Sudo, args.RequirePasswordFromStdin, config.user),
		Update:      osCommand.BuildCommandString(args.Update, args.Environment, args.Sudo, args.RequirePasswordFromStdin, config.user),
		Delete:      osCommand.BuildCommandString(args.Delete, args.Environment, args.Sudo, args.RequirePasswordFromStdin, config.user),
		Environment: args.Environment,
		Triggers:    args.Triggers,
		Stdin:       args.Stdin,
	}, nil
}

// Transformer is a function that can be used to modify the command name and args.
// Examples: swapping `args.Delete` with `args.Create`, or adding `args.Triggers`, or editing the name
type Transformer func(name string, args Args) (string, Args)

type RunnerConfiguration struct {
	user       string
	connection remote.ConnectionInput
}

type Command interface {
	pulumi.Resource

	StdoutOutput() pulumi.StringOutput
	StderrOutput() pulumi.StringOutput
}

type LocalCommand struct {
	*local.Command
}

type RemoteCommand struct {
	*remote.Command
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
	Config() RunnerConfiguration
	OsCommand() OSCommand

	Command(name string, args *Args, opts ...pulumi.ResourceOption) (Command, error)
	NewCopyFile(name string, localPath, remotePath pulumi.StringInput, opts ...pulumi.ResourceOption) (pulumi.Resource, error)
	CopyWindowsFile(name string, src, dst pulumi.StringInput, opts ...pulumi.ResourceOption) (pulumi.Resource, error)
	CopyUnixFile(name string, src, dst pulumi.StringInput, opts ...pulumi.ResourceOption) (pulumi.Resource, error)
	PulumiOptions() []pulumi.ResourceOption
}

var _ Runner = &RemoteRunner{}
var _ Runner = &LocalRunner{}

type RemoteRunner struct {
	e           config.Env
	namer       namer.Namer
	waitCommand Command
	config      RunnerConfiguration
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
		config: RunnerConfiguration{
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

func (r *RemoteRunner) Config() RunnerConfiguration {
	return r.config
}

func (r *RemoteRunner) OsCommand() OSCommand {
	return r.osCommand
}

func (r *RemoteRunner) Command(name string, args *Args, opts ...pulumi.ResourceOption) (Command, error) {
	if args.Sudo && r.config.user != "" {
		r.e.Ctx().Log.Info(fmt.Sprintf("warning: running sudo command on a runner with user %s, discarding user", r.config.user), nil)
	}

	remoteArgs, err := args.toRemoteCommandArgs(r.config, r.osCommand)
	if err != nil {
		return nil, err
	}

	cmd, err := remote.NewCommand(r.e.Ctx(), r.namer.ResourceName("cmd", name), remoteArgs, utils.MergeOptions(r.options, opts...)...)

	if err != nil {
		return nil, err
	}

	return &RemoteCommand{cmd}, nil
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
	config    RunnerConfiguration
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
		config: RunnerConfiguration{
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

func (r *LocalRunner) Config() RunnerConfiguration {
	return r.config
}

func (r *LocalRunner) OsCommand() OSCommand {
	return r.osCommand
}

func (r *LocalRunner) Command(name string, args *Args, opts ...pulumi.ResourceOption) (Command, error) {
	opts = utils.MergeOptions[pulumi.ResourceOption](opts, r.e.WithProviders(config.ProviderCommand))
	localArgs, err := args.toLocalCommandArgs(r.config, r.osCommand)
	if err != nil {
		return nil, err
	}

	cmd, err := local.NewCommand(r.e.Ctx(), r.namer.ResourceName("cmd", name), localArgs, opts...)

	if err != nil {
		return nil, err
	}

	return &LocalCommand{cmd}, nil
}

func (r *LocalRunner) NewCopyFile(name string, localPath, remotePath pulumi.StringInput, opts ...pulumi.ResourceOption) (pulumi.Resource, error) {
	return r.osCommand.NewCopyFile(r, name, localPath, remotePath, opts...)
}

func (r *LocalRunner) PulumiOptions() []pulumi.ResourceOption {
	return []pulumi.ResourceOption{}
}

func (r *LocalRunner) CopyWindowsFile(name string, src, dst pulumi.StringInput, opts ...pulumi.ResourceOption) (pulumi.Resource, error) {
	createCmd := pulumi.Sprintf("Copy-Item -Path '%v' -Destination '%v'", src, dst)
	deleteCmd := pulumi.Sprintf("Remove-Item -Path '%v'", dst)
	useSudo := false // TODO A

	return r.Command(name,
		&Args{
			Create:   createCmd,
			Delete:   deleteCmd,
			Sudo:     useSudo,
			Triggers: pulumi.Array{createCmd, deleteCmd, pulumi.BoolPtr(useSudo)},
		}, opts...)
}

func (r *LocalRunner) CopyUnixFile(name string, src, dst pulumi.StringInput, opts ...pulumi.ResourceOption) (pulumi.Resource, error) {
	createCmd := pulumi.Sprintf("cp '%v' '%v'", src, dst)
	deleteCmd := pulumi.Sprintf("rm '%v'", dst)
	useSudo := false // TODO A

	return r.Command(name,
		&Args{
			Create:   createCmd,
			Delete:   deleteCmd,
			Sudo:     useSudo,
			Triggers: pulumi.Array{createCmd, deleteCmd, pulumi.BoolPtr(useSudo)},
		}, opts...)
}

func (r *RemoteRunner) CopyWindowsFile(name string, src, dst pulumi.StringInput, opts ...pulumi.ResourceOption) (pulumi.Resource, error) {
	return remote.NewCopyFile(r.Environment().Ctx(), r.Namer().ResourceName("copy", name), &remote.CopyFileArgs{
		Connection: r.Config().connection,
		LocalPath:  src,
		RemotePath: dst,
		Triggers:   pulumi.Array{src, dst},
	}, utils.MergeOptions(r.PulumiOptions(), opts...)...)
}

func (r *RemoteRunner) CopyUnixFile(name string, src, dst pulumi.StringInput, opts ...pulumi.ResourceOption) (pulumi.Resource, error) {
	tempRemotePath := src.ToStringOutput().ApplyT(func(path string) string {
		return filepath.Join(r.OsCommand().GetTemporaryDirectory(), filepath.Base(path))
	}).(pulumi.StringOutput)

	tempCopyFile, err := remote.NewCopyFile(r.Environment().Ctx(), r.Namer().ResourceName("copy", name), &remote.CopyFileArgs{
		Connection: r.Config().connection,
		LocalPath:  src,
		RemotePath: tempRemotePath,
		Triggers:   pulumi.Array{src, tempRemotePath},
	}, utils.MergeOptions(r.PulumiOptions(), opts...)...)

	if err != nil {
		return nil, err
	}

	moveCommand, err := r.OsCommand().MoveRemoteFile(r, name, tempRemotePath, dst, true, utils.MergeOptions(opts, utils.PulumiDependsOn(tempCopyFile))...)
	if err != nil {
		return nil, err
	}

	return moveCommand, err
}
