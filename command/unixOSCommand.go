package command

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/alessio/shellescape"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	linuxTempDir = "/tmp"
)

var _ OSCommand = (*unixOSCommand)(nil)

type unixOSCommand struct{}

func NewUnixOSCommand() OSCommand {
	return unixOSCommand{}
}

func (unixOSCommand) CreateDirectory(
	runner *Runner,
	name string,
	remotePath pulumi.StringInput,
	useSudo bool,
	opts ...pulumi.ResourceOption) (*remote.Command, error) {
	return createDirectory(
		runner,
		name,
		"mkdir -p %s",
		"rm -rf %s",
		remotePath,
		useSudo,
		opts...)
}

func (unixOSCommand) CopyInlineFile(
	runner *Runner,
	name string,
	fileContent pulumi.StringInput,
	remotePath string,
	useSudo bool,
	append bool,
	opts ...pulumi.ResourceOption) (*remote.Command, error) {
	var createCmd pulumi.StringInput
	if append {
		createCmd = utils.AppendStringCommand(remotePath, useSudo)
	} else {
		createCmd = utils.WriteStringCommand(remotePath, useSudo)
	}
	return copyInlineFile(name, runner, fileContent, useSudo, createCmd, opts...)
}

func (fs unixOSCommand) GetTemporaryDirectory() string {
	return linuxTempDir
}

func (fs unixOSCommand) BuildCommandString(command pulumi.StringInput, env pulumi.StringMap, sudo bool, user string) pulumi.StringInput {
	var prefix string

	if sudo {
		prefix = "sudo"
	} else if user != "" {
		prefix = fmt.Sprintf("sudo -u %s", user)
	}

	var envVars pulumi.StringArray
	for varName, varValue := range env {
		envVars = append(envVars, pulumi.Sprintf(`%s="%s"`, varName, varValue))
	}

	return buildCommandString(command, envVars, func(envVarsStr pulumi.StringOutput) pulumi.StringInput {
		commandEscaped := command.ToStringOutput().ApplyT(func(command string) string {
			return shellescape.Quote(command)
		}).(pulumi.StringOutput)

		return pulumi.Sprintf("%s %s bash -c %s", prefix, envVarsStr, commandEscaped)
	})
}
