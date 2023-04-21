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
	opts ...pulumi.ResourceOption) (*remote.Command, error) {

	createCmd := utils.WriteStringCommand(remotePath, useSudo)
	return copyInlineFile(name, runner, fileContent, useSudo, createCmd, opts...)
}

func (fs unixOSCommand) GetTemporaryDirectory() string {
	return linuxTempDir
}

func (fs unixOSCommand) BuildCommandString(command pulumi.StringInput, env pulumi.StringMap, sudo bool, user string) pulumi.StringInput {
	formattedCommand := formatCommandIfNeeded(command, sudo, user)

	var envVars pulumi.StringArray
	for varName, varValue := range env {
		envVars = append(envVars, pulumi.Sprintf(`%s="%s"`, varName, varValue))
	}

	return buildCommandString(formattedCommand, envVars, func(envVarsStr pulumi.StringOutput) pulumi.StringInput {
		return pulumi.Sprintf("%s %s", envVarsStr, formattedCommand)
	})
}

func formatCommandIfNeeded(command pulumi.StringInput, sudo bool, user string) pulumi.StringInput {
	if !sudo && user == "" {
		return command
	}
	var formattedCommand pulumi.StringInput
	if sudo {
		formattedCommand = pulumi.Sprintf("sudo %s", command)
	} else if user != "" {
		formattedCommand = command.ToStringOutput().ApplyT(func(cmd string) string {
			return fmt.Sprintf("sudo -u %s bash -c %s", user, shellescape.Quote(cmd))
		}).(pulumi.StringOutput)
	}
	return formattedCommand
}
