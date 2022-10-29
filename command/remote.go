package command

import (
	"fmt"
	"hash/fnv"
	"path"
	"strings"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Runner struct {
	connectionName string
	connection     remote.ConnectionInput
	waitCommand    *remote.Command
}

func NewRunner(connName string, conn remote.ConnectionInput, readyFunc func(*Runner) (*remote.Command, error)) (*Runner, error) {
	runner := &Runner{
		connectionName: connName,
		connection:     conn,
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

func (r *Runner) Command(ctx *pulumi.Context, name string, create, update, delete pulumi.StringInput, env pulumi.StringMap, sudo bool, opts ...pulumi.ResourceOption) (*remote.Command, error) {
	create = r.buildCommandInput(create, env, sudo)
	update = r.buildCommandInput(update, env, sudo)
	delete = r.buildCommandInput(delete, env, sudo)

	if r.waitCommand != nil {
		opts = append(opts, pulumi.DependsOn([]pulumi.Resource{r.waitCommand}))
	}

	return remote.NewCommand(ctx, r.connectionName+"-"+name,
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

func UniqueCommandName(name string, createCmd, updateCmd, deleteCmd string) string {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(createCmd + updateCmd + deleteCmd))

	return fmt.Sprintf("%s-remote-%d", name, hash.Sum32())
}

func TempDir(ctx *pulumi.Context, name string, runner *Runner) (*remote.Command, string, error) {
	tempDir := path.Join("/tmp", name)
	c, err := runner.Command(ctx, "tmpdir-"+name, pulumi.Sprintf("mkdir -p %s", tempDir), nil, pulumi.Sprintf("rm -rf %s", tempDir), nil, false)
	return c, tempDir, err
}
