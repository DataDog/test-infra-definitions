package command

import (
	"strings"

	"github.com/DataDog/test-infra-definitions/common"
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

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

func (r *Runner) Command(name string, create, update, delete pulumi.StringInput, env pulumi.StringMap, sudo bool, opts ...pulumi.ResourceOption) (*remote.Command, error) {
	create = r.buildCommandInput(create, env, sudo)
	update = r.buildCommandInput(update, env, sudo)
	delete = r.buildCommandInput(delete, env, sudo)

	if r.waitCommand != nil {
		opts = append(opts, pulumi.DependsOn([]pulumi.Resource{r.waitCommand}))
	}

	return remote.NewCommand(r.e.Ctx, r.namer.ResourceName("cmd", name),
		&remote.CommandArgs{
			Connection: r.connection,
			Create:     create,
			Update:     update,
			Delete:     delete,
		}, opts...)
}

func (r *Runner) buildCommandInput(command pulumi.StringInput, env pulumi.StringMap, sudo bool) pulumi.StringInput {
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
