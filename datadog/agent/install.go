package agent

import (
	"path"
	"strings"

	"github.com/DataDog/test-infra-definitions/command"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/common/vm"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Installer struct {
	dependsOn pulumi.Resource
}

func NewInstaller(vm vm.VM, options ...func(*params) error) (*Installer, error) {
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

	if params.agentConfig != "" {
		fileManager := command.NewFileManager(runner)
		remotePath := path.Join(os.GetAgentConfigFolder(), "datadog.yaml")
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
	return &Installer{dependsOn: lastCommand}, err
}
