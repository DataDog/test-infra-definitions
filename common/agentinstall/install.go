package agentinstall

import (
	"strings"

	"github.com/DataDog/test-infra-definitions/command"
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/os"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Install(runner *command.Runner, env *config.CommonEnvironment, params *Params, os os.OS) (pulumi.Resource, error) {
	cmd, err := os.GetAgentInstallCmd(params.version)
	if err != nil {
		return nil, err
	}
	commonNamer := env.CommonNamer
	lastCommand, err := runner.Command(
		commonNamer.ResourceName("agent-install", utils.StrHash(cmd)),
		&command.Args{
			Create: pulumi.Sprintf(cmd, env.AgentAPIKey()),
		})
	if err != nil {
		return nil, err
	}

	if params.agentConfig != "" {
		fileManager := command.NewFileManager(runner)
		remotePath := os.GetAgentConfigPath()
		agentConfig := env.AgentAPIKey().ApplyT(func(apiKey string) pulumi.String {
			config := strings.ReplaceAll(params.agentConfig, "{{API_KEY}}", apiKey)
			return pulumi.String(config)
		}).(pulumi.StringInput)
		lastCommand, err = fileManager.CopyInlineFile("agent-config", agentConfig, remotePath, true, pulumi.DependsOn([]pulumi.Resource{lastCommand}))
		if err != nil {
			return nil, err
		}

	}

	// When the file content has changed, make sure the Agent is restarted.
	serviceManager := os.GetServiceManager()
	for _, cmd := range serviceManager.RestartAgentCmd() {
		restartAgentRes := commonNamer.ResourceName("restart-agent", utils.StrHash(cmd, params.agentConfig))
		lastCommand, err = runner.Command(
			restartAgentRes,
			&command.Args{
				Create: pulumi.String(cmd),
			}, pulumi.DependsOn([]pulumi.Resource{lastCommand}))
	}
	return lastCommand, err
}
