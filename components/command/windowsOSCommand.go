package command

import (
	"fmt"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"strings"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var _ OSCommand = (*windowsOSCommand)(nil)

type windowsOSCommand struct{}

func NewWindowsOSCommand() OSCommand {
	return windowsOSCommand{}
}

// CreateDirectory if it does not exist
func (fs windowsOSCommand) CreateDirectory(
	runner *Runner,
	name string,
	remotePath pulumi.StringInput,
	_ bool,
	opts ...pulumi.ResourceOption,
) (*remote.Command, error) {
	useSudo := false
	return createDirectory(
		runner,
		name,
		fmt.Sprintf("New-Item -Force -Path %v -ItemType Directory", remotePath),
		fmt.Sprintf("if (-not (Test-Path -Path %v/*)) { Remove-Item -Path %v -ErrorAction SilentlyContinue }", remotePath, remotePath),
		useSudo,
		opts...)
}

func (fs windowsOSCommand) GetTemporaryDirectory() string {
	return "$env:TEMP"
}

func (fs windowsOSCommand) GetHomeDirectory() string {
	// %HOMEDRIVE% returns the disk drive where home directory is located
	// %HOMEPATH% returns the path to the home directory related to HOMEDRIVE
	return "$env:HOMEDRIVE$env:HOMEPATH"
}

func (fs windowsOSCommand) BuildCommandString(
	command pulumi.StringInput,
	env pulumi.StringMap,
	_ bool,
	_ bool,
	_ string,
) pulumi.StringInput {
	var envVars pulumi.StringArray
	for varName, varValue := range env {
		envVars = append(envVars, pulumi.Sprintf(`$env:%v = '%v'; `, varName, varValue))
	}

	return buildCommandString(command, envVars, func(envVarsStr pulumi.StringOutput) pulumi.StringInput {
		return pulumi.Sprintf("%s %s", envVarsStr, command)
	})
}

func (fs windowsOSCommand) IsPathAbsolute(path string) bool {
	// valid absolute path prefixes: "x:\", "x:/", "\\", "//" ]
	if len(path) < 2 {
		return false
	}
	if strings.HasPrefix(path, "//") || strings.HasPrefix(path, `\\`) {
		return true
	} else if strings.Index(path, ":/") == 1 {
		return true
	} else if strings.Index(path, `:\`) == 1 {
		return true
	}
	return false
}

func (fs windowsOSCommand) NewCopyFile(runner *Runner, localPath, remotePath string, opts ...pulumi.ResourceOption) (*remote.CopyFile, error) {
	return remote.NewCopyFile(runner.e.Ctx(), runner.namer.ResourceName("copy", remotePath), &remote.CopyFileArgs{
		Connection: runner.config.connection,
		LocalPath:  pulumi.String(localPath),
		RemotePath: pulumi.String(remotePath),
		Triggers:   pulumi.Array{pulumi.String(localPath), pulumi.String(remotePath)},
	}, utils.MergeOptions(runner.options, opts...)...)
}
