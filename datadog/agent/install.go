package agent

import (
	"fmt"
	"path"
	"strings"

	"github.com/DataDog/test-infra-definitions/command"
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/namer"
	"github.com/DataDog/test-infra-definitions/common/os"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/common/vm"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Installer struct {
	dependsOn pulumi.Resource
}

// Temporary requires vm.UnixVM until FileManager is available in VM
func NewInstaller(vm *vm.UnixVM, options ...func(*params) error) (*Installer, error) {
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

	var configHash string
	lastCommand, configHash, err = updateAgentConfig(
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
		lastCommand, err = runner.Command(
			restartAgentRes,
			&command.Args{
				Create:   pulumi.String(cmd),
				Triggers: pulumi.Array{pulumi.String(utils.StrHash(cmd, configHash, integsHash))},
			}, utils.PulumiDependsOn(lastCommand))
	}
	return &Installer{dependsOn: lastCommand}, err
}

func updateAgentConfig(
	namer namer.Namer,
	fileManager *command.FileManager,
	env *config.CommonEnvironment,
	agentConfig string,
	extraAgentConfig []string,
	os os.OS,
	lastCommand *remote.Command) (*remote.Command, string, error) {
	agentConfigFullPath := path.Join(os.GetAgentConfigFolder(), "datadog.yaml")
	var err error
	var parts = []string{agentConfig}
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
			return nil, "", err
		}
	}

	for _, extraConfig := range extraAgentConfig {
		parts = append(parts, extraConfig)
		lastCommand, err = fileManager.AppendInlineFile(
			namer.ResourceName("config-append"),
			pulumi.String(fmt.Sprintf("\n%v\n", extraConfig)),
			agentConfigFullPath,
			true,
			utils.PulumiDependsOn(lastCommand))
		if err != nil {
			return nil, "", err
		}
	}
	return lastCommand, utils.StrHash(parts...), nil
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
