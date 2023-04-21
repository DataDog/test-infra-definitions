package command

import (
	"fmt"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var _ OSCommand = (*windowsOSCommand)(nil)

type windowsOSCommand struct{}

func NewWindowsOSCommand() OSCommand {
	return windowsOSCommand{}
}

func (fs windowsOSCommand) CreateDirectory(
	runner *Runner,
	name string,
	remotePath pulumi.StringInput,
	_ bool,
	opts ...pulumi.ResourceOption) (*remote.Command, error) {
	useSudo := false
	return createDirectory(
		runner,
		name,
		"New-Item -Path %s -ItemType Directory -Force",
		"Remove-Item -Path %s -Force -ErrorAction SilentlyContinue -Recurse",
		remotePath,
		useSudo,
		opts...)
}

func (fs windowsOSCommand) CopyInlineFile(
	runner *Runner,
	fileContent pulumi.StringInput,
	remotePath string,
	useSudo bool,
	opts ...pulumi.ResourceOption) (*remote.Command, error) {
	backupPath := remotePath + "." + backupExtension
	backupCmd := fmt.Sprintf("if (Test-Path -Path '%v') { Move-Item -Force -Path '%v' -Destination '%v'}", remotePath, remotePath, backupPath)
	createCmd := fmt.Sprintf(`%v; [System.Console]::In.ReadToEnd() | Out-File -FilePath '%v'`, backupCmd, remotePath)

	deleteMoveCmd := fmt.Sprintf(`Move-Item -Force -Path '%v' -Destination '%v'`, backupPath, remotePath)
	deleteRemoveCmd := fmt.Sprintf(`Remove-Item -Force -Path '%v'`, remotePath)
	deleteCmd := fmt.Sprintf("if (Test-Path -Path '%v') { %v } else { %v }", backupPath, deleteMoveCmd, deleteRemoveCmd)
	return copyInlineFile(remotePath, runner, fileContent, useSudo, createCmd, deleteCmd, opts...)
}

func (fs windowsOSCommand) GetTemporaryDirectory() string {
	return "$env:TEMP"
}

func (fs windowsOSCommand) BuildCommandString(
	command pulumi.StringInput,
	env pulumi.StringMap,
	_ bool,
	_ string) pulumi.StringInput {
	var envVars pulumi.StringArray
	for varName, varValue := range env {
		envVars = append(envVars, pulumi.Sprintf(`$env:%v = '%v'; `, varName, varValue))
	}

	return buildCommandString(command, envVars, func(envVarsStr pulumi.StringOutput) pulumi.StringInput {
		return pulumi.Sprintf("%s %s", envVarsStr, command)
	})
}
