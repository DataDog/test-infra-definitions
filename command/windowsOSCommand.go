package command

import (
	"path"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var _ osCommand = (*windowsOSCommand)(nil)

type windowsOSCommand struct{}

func newWindowsOSCommand() osCommand {
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
	append bool,
	opts ...pulumi.ResourceOption) (*remote.Command, error) {
	flags := ""
	if append {
		flags = "-Append"
	}

	createCmd := pulumi.Sprintf(`[System.Console]::In.ReadToEnd() | Out-File %v -FilePath %v`, flags, remotePath)
	return copyInlineFile(name, runner, fileContent, useSudo, createCmd, opts...)
}

func (fs windowsOSCommand) CreateTemporaryFolder(
	runner *Runner,
	resourceName string,
	opts ...pulumi.ResourceOption) (*remote.Command, string, error) {
	tempDir := path.Join("$env:TEMP", resourceName)
	folderCmd, err := fs.CreateDirectory(runner, "create-temporary-folder-"+resourceName, pulumi.String(tempDir), false, opts...)
	return folderCmd, tempDir, err
}

func (fs windowsOSCommand) BuildCommand(
	command pulumi.StringInput,
	env pulumi.StringMap,
	_ bool,
	_ string) pulumi.StringInput {
	var envVars pulumi.StringArray
	for varName, varValue := range env {
		envVars = append(envVars, pulumi.Sprintf(`$env:%v = '%v'; `, varName, varValue))
	}

	return buildCommand(command, envVars, func(envVarsStr pulumi.StringOutput) pulumi.StringInput {
		return pulumi.Sprintf("%s %s", envVarsStr, command)
	})
}
