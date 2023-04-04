package command

import (
	"fmt"
	"path"

	"github.com/alessio/shellescape"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	linuxTempDir = "/tmp"
)

var _ osCommand = (*unixOSCommand)(nil)

type unixOSCommand struct{}

func newUnixOSCommand() osCommand {
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
	fileContent pulumi.StringInput,
	remotePath string,
	useSudo bool,
	opts ...pulumi.ResourceOption) (*remote.Command, error) {

	sudo := ""
	if useSudo {
		sudo = "sudo"
	}
	backupPath := remotePath + "." + backupExtension
	backupCmd := fmt.Sprintf("if [ -f '%v' ]; then %v mv -f '%v' '%v'; fi", remotePath, sudo, remotePath, backupPath)
	createCmd := fmt.Sprintf("(%v) && cat - | %s tee %s > /dev/null", backupCmd, sudo, remotePath)
	deleteCmd := fmt.Sprintf("if [ -f '%v' ]; then %v mv -f '%v' '%v'; else %v rm -f '%v'; fi", backupPath, sudo, backupPath, remotePath, sudo, remotePath)
	return copyInlineFile(remotePath, runner, fileContent, useSudo, createCmd, deleteCmd, opts...)
}

func (fs unixOSCommand) CreateTemporaryFolder(runner *Runner, resourceName string, opts ...pulumi.ResourceOption) (*remote.Command, string, error) {
	tempDir := path.Join(linuxTempDir, resourceName)
	folderCmd, err := fs.CreateDirectory(runner, "create-temporary-folder-"+resourceName, pulumi.String(tempDir), false, opts...)
	return folderCmd, tempDir, err
}

func (fs unixOSCommand) BuildCommand(command pulumi.StringInput, env pulumi.StringMap, sudo bool, user string) pulumi.StringInput {
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

	return buildCommand(command, envVars, func(envVarsStr pulumi.StringOutput) pulumi.StringInput {
		commandEscaped := command.ToStringOutput().ApplyT(func(command string) string {
			return shellescape.Quote(command)
		}).(pulumi.StringOutput)

		return pulumi.Sprintf("%s %s bash -c %s", prefix, envVarsStr, commandEscaped)
	})
}
