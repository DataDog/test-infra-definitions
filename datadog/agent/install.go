package agent

import (
	"path"
	"strings"

	"github.com/DataDog/test-infra-definitions/command"
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/namer"
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

	var updateConfigTriggers pulumi.StringArrayInput
	lastCommand, updateConfigTriggers, err = updateAgentConfig(
		commonNamer,
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
	lastCommand, integsHash, err = installIntegrations(commonNamer, vm.GetFileManager(), params.integrations, os, lastCommand)

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
				Triggers: pulumi.Array{cmdPulumiStr, updateConfigTriggers, pulumi.String(integsHash)},
			}, utils.PulumiDependsOn(lastCommand))
	}
	return &Installer{dependsOn: lastCommand, vm: vm}, err
}

func updateAgentConfig(
	namer namer.Namer,
	fileManager *command.FileManager,
	env *config.CommonEnvironment,
	agentConfig string,
	extraAgentConfig []pulumi.StringInput,
	os os.OS,
	lastCommand *remote.Command) (*remote.Command, pulumi.StringArrayInput, error) {
	agentConfigFullPath := path.Join(os.GetAgentConfigFolder(), "datadog.yaml")
	var err error
	var parts = []pulumi.StringInput{pulumi.String(agentConfig)}
	if agentConfig != "" {
		agentConfigWithAPIKEY := env.AgentAPIKey().ApplyT(func(apiKey string) pulumi.String {
			config := strings.ReplaceAll(agentConfig, "{{API_KEY}}", apiKey)
			return pulumi.String(config)
		}).(pulumi.StringInput)
		lastCommand, err = fileManager.CopyInlineFile(
			namer.ResourceName("agent-config"),
			agentConfigWithAPIKEY,
			agentConfigFullPath,
			true,
			utils.PulumiDependsOn(lastCommand))
		if err != nil {
			return nil, pulumi.StringArray{}, err
		}
	}

	for _, extraConfig := range extraAgentConfig {
		parts = append(parts, extraConfig)
		lastCommand, err = fileManager.AppendInlineFile(
			namer.ResourceName("config-append"),
			pulumi.Sprintf("\n%v\n", extraConfig),
			agentConfigFullPath,
			true,
			utils.PulumiDependsOn(lastCommand))
		if err != nil {
			return nil, pulumi.StringArray{}, err
		}
	}
	return lastCommand, pulumi.StringArray(parts), nil
}

func installIntegrations(
	namer namer.Namer,
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
			namer.ResourceName("integration", folderName),
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
