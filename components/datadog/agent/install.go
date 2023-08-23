package agent

import (
	"fmt"
	"path"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/namer"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/components/datadog/agentparams"
	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/components/vm"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var _ utils.RemoteServiceDeserializer[ClientData] = (*Installer)(nil)

// Installer is an installer for the Agent on a virtual machine
type Installer struct {
	dependsOn pulumi.Resource
	vm        vm.VM
}

// NewInstaller creates a new instance of [*Installer]
func NewInstaller(vm vm.VM, options ...agentparams.Option) (*Installer, error) {
	env := vm.GetCommonEnvironment()
	params, err := agentparams.NewParams(env, options...)
	if err != nil {
		return nil, err
	}

	os := vm.GetOS()
	cmd, err := os.GetAgentInstallCmd(params.Version)
	if err != nil {
		return nil, err
	}
	commonNamer := env.CommonNamer
	runner := vm.GetRunner()
	lastCommand, err := runner.Command(
		commonNamer.ResourceName("agent-install"),
		&command.Args{
			Create:   pulumi.Sprintf(cmd, env.AgentAPIKey()),
			Triggers: pulumi.Array{pulumi.String(cmd)},
		})
	if err != nil {
		return nil, err
	}

	var updateAgentConfigTrigger, updateSystemConfigTrigger, updateSecurityAgentConfigTrigger pulumi.StringInput
	lastCommand, updateAgentConfigTrigger, err = updateAgentConfig(
		vm.GetFileManager(),
		env,
		params.AgentConfig,
		params.ExtraAgentConfig,
		os,
		lastCommand)
	if err != nil {
		return nil, err
	}

	var filesHash string
	lastCommand, filesHash, err = installIntegrationsAndFiles(vm.GetFileManager(), params.Integrations, params.Files, os, runner, commonNamer, lastCommand)

	if err != nil {
		return nil, err
	}

	// When the config file content or the integration changed, restart the Agent.
	lastCommand, err = restartAgent(
		func(cmd pulumi.String) *command.Args {
			return &command.Args{
				Create:   cmd,
				Triggers: pulumi.Array{updateConfigTrigger, pulumi.String(filesHash)},
			}
		},
		os, runner, commonNamer.ResourceName("restart-agent"), lastCommand)

	return &Installer{dependsOn: lastCommand, vm: vm}, err
}

func restartAgent(
	argsFactory func(cmd pulumi.String) *command.Args,
	os os.OS,
	runner *command.Runner,
	resourceName string,
	lastCommand *remote.Command) (*remote.Command, error) {

	serviceManager := os.GetServiceManager()
	var err error
	for _, cmd := range serviceManager.RestartAgentCmd() {
		cmdPulumiStr := pulumi.String(cmd)
		args := argsFactory(cmdPulumiStr)
		lastCommand, err = runner.Command(
			resourceName,
			args, utils.PulumiDependsOn(lastCommand))
		if err != nil {
			return nil, err
		}
	}
	return lastCommand, nil
}

func updateAgentConfig(
	fileManager *command.FileManager,
	env *config.CommonEnvironment,
	agentConfig string,
	extraAgentConfig []pulumi.StringInput,
	os os.OS,
	lastCommand *remote.Command) (*remote.Command, pulumi.StringInput, error) {

	agentConfigFullPath := path.Join(os.GetAgentConfigFolder(), "datadog.yaml")
	var err error

	pulumiAgentString := pulumi.String(agentConfig).ToStringOutput()
	for _, extraConfig := range extraAgentConfig {
		pulumiAgentString = pulumi.Sprintf("%v\n%v", pulumiAgentString, extraConfig)
	}

	agentConfigWithAPIKEY := pulumi.Sprintf("api_key: %v\n%v", env.AgentAPIKey(), pulumiAgentString)
	lastCommand, err = fileManager.CopyInlineFile(
		agentConfigWithAPIKEY,
		agentConfigFullPath,
		true,
		utils.PulumiDependsOn(lastCommand))
	if err != nil {
		return nil, pulumiAgentString, err
	}

	return lastCommand, pulumiAgentString, nil
}

func installIntegrationsAndFiles(
	fileManager *command.FileManager,
	integrations map[string]*agentparams.FileDefinition,
	files map[string]*agentparams.FileDefinition,
	os os.OS,
	runner *command.Runner,
	commonNamer namer.Namer,
	lastCommand *remote.Command) (*remote.Command, string, error) {
	var parts []string
	for filePath, fileDef := range integrations {
		parts = append(parts, filePath, fileDef.Content)
	}
	integrationHash := utils.StrHash(parts...)
	var err error
	// When an integration is disabled, its configuration is removed during the pulumi delete action.
	// Agent must be restarted.
	// As delete actions are executed in reverse order based on dependency, it is not possible
	// to use the same restart command as the one used when enabling an integration.
	lastCommand, err = restartAgent(
		func(cmd pulumi.String) *command.Args {
			return &command.Args{
				Delete:   cmd,
				Triggers: pulumi.Array{pulumi.String(integrationHash)},
			}
		},
		os, runner, commonNamer.ResourceName("restart-agent-integration-removal"), lastCommand)
	if err != nil {
		return nil, "", err
	}

	// filePath is absolute path from params.WithFile but relative from params.WithIntegration
	for filePath, fileDef := range integrations {
		configFolder := os.GetAgentConfigFolder()
		fullPath := path.Join(configFolder, filePath)

		lastCommand, err = writeFileDefinition(fileManager, fullPath, fileDef.Content, fileDef.UseSudo, lastCommand)
		if err != nil {
			return nil, "", err
		}
	}

	for fullPath, fileDef := range files {
		if !os.CheckIsAbsPath(fullPath) {
			return nil, "", fmt.Errorf("failed to write file: \"%s\" is not an absolute filepath", fullPath)
		}

		lastCommand, err = writeFileDefinition(fileManager, fullPath, fileDef.Content, fileDef.UseSudo, lastCommand)
		if err != nil {
			return nil, "", err
		}

		parts = append(parts, fullPath, fileDef.Content)
	}

	return lastCommand, utils.StrHash(parts...), nil
}

func writeFileDefinition(
	fileManager *command.FileManager,
	fullPath string,
	content string,
	useSudo bool,
	lastCommand *remote.Command) (*remote.Command, error) {

	var err error
	folderPath, _ := path.Split(fullPath)

	// create directory, if it does not exist
	lastCommand, err = fileManager.CreateDirectory(fullPath, pulumi.String(folderPath), useSudo, utils.PulumiDependsOn(lastCommand))
	if err != nil {
		return nil, err
	}
	lastCommand, err = fileManager.CopyInlineFile(
		pulumi.String(content),
		fullPath, useSudo, utils.PulumiDependsOn(lastCommand))
	if err != nil {
		return nil, err
	}
	return lastCommand, nil
}

type ClientData struct {
	Connection utils.Connection
}

func (installer *Installer) Deserialize(result auto.UpResult) (*ClientData, error) {
	vmData, err := installer.vm.Deserialize(result)
	if err != nil {
		return nil, err
	}

	return &ClientData{Connection: vmData.Connection}, nil
}

func (installer *Installer) VM() vm.VM {
	return installer.vm
}
