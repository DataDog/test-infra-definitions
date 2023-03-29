package command

import (
	"fmt"
	"strings"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/namer"
	"github.com/alessio/shellescape"
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

func (args *Args) toRemoteCommandArgs(config runnerConfiguration) *remote.CommandArgs {
	var prefix string

	if args.Sudo {
		prefix = "sudo"
	} else if config.user != "" {
		prefix = fmt.Sprintf("sudo -u %s", config.user)
	}

	return &remote.CommandArgs{
		Connection: config.connection,
		Create:     args.buildCommandInput(args.Create, args.Environment, prefix),
		Update:     args.buildCommandInput(args.Update, args.Environment, prefix),
		Delete:     args.buildCommandInput(args.Delete, args.Environment, prefix),
		Triggers:   args.Triggers,
		Stdin:      args.Stdin,
	}
}

func (args *Args) buildCommandInput(command pulumi.StringInput, env pulumi.StringMap, prefix string) pulumi.StringInput {
	if command == nil {
		return nil
	}

	var envVars pulumi.StringArray
	for varName, varValue := range env {
		envVars = append(envVars, pulumi.Sprintf(`%s="%s"`, varName, varValue))
	}

	envVarsStr := envVars.ToStringArrayOutput().ApplyT(func(inputs []string) string {
		return strings.Join(inputs, " ")
	}).(pulumi.StringOutput)

	commandEscaped := command.ToStringOutput().ApplyT(func(command string) string {
		return shellescape.Quote(command)
	}).(pulumi.StringOutput)

	return pulumi.Sprintf("%s %s bash -c %s", prefix, envVarsStr, commandEscaped)
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
}

func WithUser(user string) func(*Runner) {
	return func(r *Runner) {
		r.config.user = user
	}
}

func NewRunner(e config.CommonEnvironment, connName string, conn remote.ConnectionInput, readyFunc func(*Runner) (*remote.Command, error), options ...func(*Runner)) (*Runner, error) {
	runner := &Runner{
		e:     e,
		namer: namer.NewNamer(e.Ctx, "remote").WithPrefix(connName),
		config: runnerConfiguration{
			connection: conn,
		},
	}

	for _, opt := range options {
		opt(runner)
	}

	var err error
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

	return remote.NewCommand(r.e.Ctx, r.namer.ResourceName("cmd", name), args.toRemoteCommandArgs(r.config), opts...)
}

type LocalRunner struct {
	e      config.CommonEnvironment
	namer  namer.Namer
	config runnerConfiguration
}

func WithLocalUser(user string) func(*LocalRunner) {
	return func(l *LocalRunner) {
		l.config.user = user
	}
}

func NewLocalRunner(e config.CommonEnvironment, options ...func(*LocalRunner)) *LocalRunner {
	localRunner := &LocalRunner{
		e:     e,
		namer: namer.NewNamer(e.Ctx, "local"),
	}

	for _, opt := range options {
		opt(localRunner)
	}

	return localRunner
}

func (args *Args) toLocalCommandArgs(config runnerConfiguration) *local.CommandArgs {
	var prefix string
	if args.Sudo {
		prefix = "sudo"
	} else if config.user != "" {
		prefix = fmt.Sprintf("sudo -u %s", config.user)
	}

	return &local.CommandArgs{
		Create:   args.buildCommandInput(args.Create, args.Environment, prefix),
		Update:   args.buildCommandInput(args.Update, args.Environment, prefix),
		Delete:   args.buildCommandInput(args.Delete, args.Environment, prefix),
		Triggers: args.Triggers,
		Stdin:    args.Stdin,
	}
}

func (r *LocalRunner) Command(name string, args *Args, opts ...pulumi.ResourceOption) (*local.Command, error) {
	return local.NewCommand(r.e.Ctx, r.namer.ResourceName("cmd", name), args.toLocalCommandArgs(r.config), opts...)
}
