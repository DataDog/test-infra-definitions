package command

import (
	"strings"

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

func (args *Args) toRemoteCommandArgs(c remote.ConnectionInput) *remote.CommandArgs {
	return &remote.CommandArgs{
		Connection: c,
		Create:     args.buildCommandInput(args.Create, args.Environment, args.Sudo),
		Update:     args.buildCommandInput(args.Update, args.Environment, args.Sudo),
		Delete:     args.buildCommandInput(args.Delete, args.Environment, args.Sudo),
		Triggers:   args.Triggers,
		Stdin:      args.Stdin,
	}
}

func (args *Args) buildCommandInput(command pulumi.StringInput, env pulumi.StringMap, sudo bool) pulumi.StringInput {
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

	var prefix string
	if sudo {
		prefix = "sudo"
	}

	return pulumi.Sprintf("%s %s %s", prefix, envVarsStr, command)
}

type Runner struct {
	e           config.CommonEnvironment
	namer       namer.Namer
	connection  remote.ConnectionInput
	waitCommand *remote.Command
}

func NewRunner(e config.CommonEnvironment, connName string, conn remote.ConnectionInput, readyFunc func(*Runner) (*remote.Command, error)) (*Runner, error) {
	runner := &Runner{
		e:          e,
		namer:      namer.NewNamer(e.Ctx, "remote").WithPrefix(connName),
		connection: conn,
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

	return remote.NewCommand(r.e.Ctx, r.namer.ResourceName("cmd", name), args.toRemoteCommandArgs(r.connection), opts...)
}

type LocalRunner struct {
	e     config.CommonEnvironment
	namer namer.Namer
}

func NewLocalRunner(e config.CommonEnvironment) *LocalRunner {
	return &LocalRunner{
		e:     e,
		namer: namer.NewNamer(e.Ctx, "local"),
	}
}

func (args *Args) toLocalCommandArgs() *local.CommandArgs {
	return &local.CommandArgs{
		Create:   args.buildCommandInput(args.Create, args.Environment, args.Sudo),
		Update:   args.buildCommandInput(args.Update, args.Environment, args.Sudo),
		Delete:   args.buildCommandInput(args.Delete, args.Environment, args.Sudo),
		Triggers: args.Triggers,
		Stdin:    args.Stdin,
	}
}

func (r *LocalRunner) Command(name string, args *Args, opts ...pulumi.ResourceOption) (*local.Command, error) {
	return local.NewCommand(r.e.Ctx, r.namer.ResourceName("cmd", name), args.toLocalCommandArgs(), opts...)
}
