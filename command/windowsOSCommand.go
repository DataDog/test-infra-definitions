package command

import (
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
	name string,
	fileContent pulumi.StringInput,
	remotePath string,
	useSudo bool,
	opts ...pulumi.ResourceOption) (*remote.Command, error) {
	createCmd := pulumi.Sprintf(`[System.Console]::In.ReadToEnd() | Out-File -FilePath %v`, remotePath)
	return copyInlineFile(name, runner, fileContent, useSudo, createCmd, opts...)
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
