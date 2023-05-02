package agent

import (
	"path"

	"github.com/DataDog/test-infra-definitions/command"
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/os"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/common/vm"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Installer struct {
	dependsOn pulumi.Resource
	vm        vm.VM
}

func NewInstaller(vm vm.VM, options ...func(*Params) error) (*Installer, error) {
	env := vm.GetCommonEnvironment()
	params, err := newParams(env, options...)
	if err != nil {
		return nil, err
	}

	os := vm.GetOS()
	cmd, err := os.GetAgentInstallCmd(params.version)
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
		params.agentConfig,
		params.extraAgentConfig,
		os,
		lastCommand)
	if err != nil {
		return nil, err
	}

	var integsHash string
	lastCommand, integsHash, err = installIntegrations(vm.GetFileManager(), params.integrations, os, lastCommand)

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
				Triggers: pulumi.Array{cmdPulumiStr, updateConfigTrigger, pulumi.String(integsHash)},
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
	if agentConfig == "" || len(extraAgentConfig) == 0 {
		// no update in agent config, safely early return
		return nil, nil, nil
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

func installIntegrations(
	fileManager *command.FileManager,
	integrations map[string]string,
	os os.OS,
	lastCommand *remote.Command) (*remote.Command, string, error) {
	configFolder := os.GetAgentConfigFolder()
	var parts []string
	for folderName, content := range integrations {
		var err error
		confPath := path.Join(configFolder, "conf.d", folderName, "conf.yaml")
		lastCommand, err = fileManager.CopyInlineFile(
			pulumi.String(content),
			confPath, true, utils.PulumiDependsOn(lastCommand))
		if err != nil {
			return nil, "", err
		}
		parts = append(parts, folderName, content)
	}

	return lastCommand, utils.StrHash(parts...), nil
}

type ClientData struct {
	Connection utils.Connection
}

func (installer *Installer) GetClientDataDeserializer() func(auto.UpResult) (*ClientData, error) {
	vmDataDeserializer := installer.vm.GetClientDataDeserializer()
	return func(result auto.UpResult) (*ClientData, error) {
		vmData, err := vmDataDeserializer(result)

		if err != nil {
			return nil, err
		}

		return &ClientData{Connection: vmData.Connection}, nil
	}
}
