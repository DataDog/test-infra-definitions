package agent

import (
	"path"

	"github.com/DataDog/test-infra-definitions/common/config"
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
		commonNamer.ResourceName("agent-install", utils.StrHash(cmd)),
		&command.Args{
			Create: pulumi.Sprintf(cmd, env.AgentAPIKey()),
		})
	if err != nil {
		return nil, err
	}

	var updateConfigTrigger pulumi.StringInput
	lastCommand, updateConfigTrigger, err = updateAgentConfig(
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
	lastCommand, filesHash, err = writeFileDefinitions(vm.GetFileManager(), params.Integrations, params.Files, os, lastCommand)

	if err != nil {
		return nil, err
	}

	// When the file content has changed, make sure the Agent is restarted.
	serviceManager := os.GetServiceManager()
	for _, cmd := range serviceManager.RestartAgentCmd() {
		restartAgentRes := commonNamer.ResourceName("restart-agent")
		cmdPulumiStr := pulumi.String(cmd)
		lastCommand, err = runner.Command(
			restartAgentRes,
			&command.Args{
				Create:   cmdPulumiStr,
				Triggers: pulumi.Array{cmdPulumiStr, updateConfigTrigger, pulumi.String(filesHash)},
			}, utils.PulumiDependsOn(lastCommand))
	}
	return &Installer{dependsOn: lastCommand, vm: vm}, err
}

func updateAgentConfig(
	fileManager *command.FileManager,
	env *config.CommonEnvironment,
	agentConfig string,
	extraAgentConfig []pulumi.StringInput,
	os os.OS,
	lastCommand *remote.Command) (*remote.Command, pulumi.StringInput, error) {
	if agentConfig == "" && len(extraAgentConfig) == 0 {
		// no update in agent config, safely early return
		return lastCommand, nil, nil
	}

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

func writeFileDefinitions(
	fileManager *command.FileManager,
	integrations map[string]string,
	files map[string]string,
	os os.OS,
	lastCommand *remote.Command) (*remote.Command, string, error) {
	var parts []string
	var err error
	for filePath, content := range integrations {
		useSudo := true
		configFolder := os.GetAgentConfigFolder()
		fullPath := path.Join(configFolder, filePath)

		folderPath, _ := path.Split(fullPath)

		// create directory, if it does not exist
		lastCommand, err = fileManager.CreateDirectory(filePath, pulumi.String(folderPath), useSudo, utils.PulumiDependsOn(lastCommand))
		if err != nil {
			return nil, "", err
		}
		lastCommand, err = fileManager.CopyInlineFile(
			pulumi.String(content),
			fullPath, useSudo, utils.PulumiDependsOn(lastCommand))
		if err != nil {
			return nil, "", err
		}
		parts = append(parts, filePath, content)
	}

	for fullPath, content := range files {
		useSudo := false
		// filePath is absolute path from params.WithFile but relative from params.WithIntegration
		folderPath, _ := path.Split(fullPath)

		// create directory, if it does not exist
		lastCommand, err = fileManager.CreateDirectory(fullPath, pulumi.String(folderPath), useSudo, utils.PulumiDependsOn(lastCommand))
		if err != nil {
			return nil, "", err
		}
		lastCommand, err = fileManager.CopyInlineFile(
			pulumi.String(content),
			fullPath, useSudo, utils.PulumiDependsOn(lastCommand))
		if err != nil {
			return nil, "", err
		}
		parts = append(parts, fullPath, content)
	}

	return lastCommand, utils.StrHash(parts...), nil
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
