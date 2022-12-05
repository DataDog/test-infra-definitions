package command

import (
	"strings"

	"github.com/DataDog/test-infra-definitions/common"
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type CommandArgs struct {
	Create      pulumi.StringInput
	Update      pulumi.StringInput
	Delete      pulumi.StringInput
	Triggers    pulumi.ArrayInput
	Environment pulumi.StringMap
	Sudo        bool
}

func (args *CommandArgs) toRemoteCommandArgs(c remote.ConnectionInput) *remote.CommandArgs {
	return &remote.CommandArgs{
		Connection: c,
		Create:     args.buildCommandInput(args.Create, args.Environment, args.Sudo),
		Update:     args.buildCommandInput(args.Update, args.Environment, args.Sudo),
		Delete:     args.buildCommandInput(args.Delete, args.Environment, args.Sudo),
		Triggers:   args.Triggers,
	}
}

func (args *CommandArgs) buildCommandInput(command pulumi.StringInput, env pulumi.StringMap, sudo bool) pulumi.StringInput {
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
	namer       common.Namer
	connection  remote.ConnectionInput
	waitCommand *remote.Command
}

func NewRunner(e config.CommonEnvironment, connName string, conn remote.ConnectionInput, readyFunc func(*Runner) (*remote.Command, error)) (*Runner, error) {
	runner := &Runner{
		e:          e,
		namer:      common.NewNamer(e.Ctx, "remote").WithPrefix(connName),
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

func (r *Runner) Command(name string, args *CommandArgs, opts ...pulumi.ResourceOption) (*remote.Command, error) {
	if r.waitCommand != nil {
		opts = append(opts, pulumi.DependsOn([]pulumi.Resource{r.waitCommand}))
	}

	return remote.NewCommand(r.e.Ctx, r.namer.ResourceName("cmd", name), args.toRemoteCommandArgs(r.connection), opts...)
}
