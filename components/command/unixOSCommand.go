package command

import (
	"fmt"

	"github.com/alessio/shellescape"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	linuxTempDir = "/tmp"
	linuxHomeDir = "$HOME"
)

var _ OSCommand = (*unixOSCommand)(nil)

type unixOSCommand struct{}

func NewUnixOSCommand() OSCommand {
	return unixOSCommand{}
}

// CreateDirectory if it does not exist
func (unixOSCommand) CreateDirectory(
	runner *Runner,
	name string,
	remotePath pulumi.StringInput,
	useSudo bool,
	opts ...pulumi.ResourceOption) (*remote.Command, error) {

	createCmd := fmt.Sprintf("mkdir -p %v", remotePath)
	deleteCmd := fmt.Sprintf(`bash -c 'if [ -z "$(ls -A %v)" ]; then rm -d %v; fi'`, remotePath, remotePath)
	// check if directory already exist
	return createDirectory(
		runner,
		name,
		createCmd,
		deleteCmd,
		useSudo,
		opts...)
}

func (unixOSCommand) CopyInlineFile(
	runner *Runner,
	fileContent pulumi.StringInput,
	remotePath string,
	useSudo bool,
	opts ...pulumi.ResourceOption) (*remote.Command, error) {

	backupPath := remotePath + "." + backupExtension
	backupCmd := fmt.Sprintf("if [ -f '%v' ]; then mv -f '%v' '%v'; fi", remotePath, remotePath, backupPath)
	createCmd := fmt.Sprintf("bash -c '(%v) && cat - | tee %v > /dev/null'", backupCmd, remotePath)
	deleteCmd := fmt.Sprintf("bash -c 'if [ -f '%v' ]; then mv -f '%v' '%v'; else rm -f '%v'; fi'", backupPath, backupPath, remotePath, remotePath)
	opts = append(opts, pulumi.ReplaceOnChanges([]string{"*"}), pulumi.DeleteBeforeReplace(true))
	return copyInlineFile(remotePath, runner, fileContent, useSudo, createCmd, deleteCmd, opts...)
}

func (fs unixOSCommand) GetTemporaryDirectory() string {
	return linuxTempDir
}

func (fs unixOSCommand) GetHomeDirectory() string {
	return linuxHomeDir
}

// BuildCommandString properly format the command string
// command can be nil
func (fs unixOSCommand) BuildCommandString(command pulumi.StringInput, env pulumi.StringMap, sudo bool, password bool, user string) pulumi.StringInput {
	formattedCommand := formatCommandIfNeeded(command, sudo, password, user)

	var envVars pulumi.StringArray
	for varName, varValue := range env {
		envVars = append(envVars, pulumi.Sprintf(`%v="%v"`, varName, varValue))
	}

	return buildCommandString(formattedCommand, envVars, func(envVarsStr pulumi.StringOutput) pulumi.StringInput {
		return pulumi.Sprintf("%s %s", envVarsStr, formattedCommand)
	})
}

func formatCommandIfNeeded(command pulumi.StringInput, sudo bool, password bool, user string) pulumi.StringInput {
	if command == nil {
		return nil
	}

	if !sudo && user == "" {
		return command
	}
	var formattedCommand pulumi.StringInput
	if sudo && password {
		formattedCommand = pulumi.Sprintf("sudo -S %v", command)
	} else if sudo {
		formattedCommand = pulumi.Sprintf("sudo %v", command)
	} else if user != "" {
		formattedCommand = command.ToStringOutput().ApplyT(func(cmd string) string {
			return fmt.Sprintf("sudo -u %v bash -c %v", user, shellescape.Quote(cmd))
		}).(pulumi.StringOutput)
	}
	return formattedCommand
}
