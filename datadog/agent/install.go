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
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Installer struct {
	dependsOn pulumi.Resource
}

// Temporary requires vm.UnixLikeVM until FileManager is available in VM
func NewInstaller(vm *vm.UnixLikeVM, options ...func(*params) error) (*Installer, error) {
	env := vm.GetCommonEnvironment()
	options = addTelemetry(options)
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
	lastCommand, configHash, err = updateAgentConfig(runner, env, params.agentConfig, os, lastCommand)
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
		restartAgentRes := commonNamer.ResourceName("restart-agent", utils.StrHash(cmd, configHash, integsHash))
		lastCommand, err = runner.Command(
			restartAgentRes,
			&command.Args{
				Create: pulumi.String(cmd),
			}, pulumi.DependsOn([]pulumi.Resource{lastCommand}))
	}
	return &Installer{dependsOn: lastCommand}, err
}

func addTelemetry(options []func(*params) error) []func(*params) error {
	config := `
instances:
  - expvar_url: http://localhost:5000/debug/vars
    max_returned_metrics: 1000
    metrics:      
      - path: ".*"
      - path: ".*/.*"
      - path: ".*/.*/.*"
`
	options = append(options, WithIntegration("go_expvar.d", config))
	return options
}

func updateAgentConfig(
	runner *command.Runner,
	env *config.CommonEnvironment,
	agentConfig string,
	os os.OS,
	lastCommand *remote.Command) (*remote.Command, string, error) {
	if agentConfig != "" {
		fileManager := command.NewFileManager(runner)
		remotePath := path.Join(os.GetAgentConfigFolder(), "datadog.yaml")
		agentConfig := env.AgentAPIKey().ApplyT(func(apiKey string) pulumi.String {
			config := strings.ReplaceAll(agentConfig, "{{API_KEY}}", apiKey)
			return pulumi.String(config)
		}).(pulumi.StringInput)
		var err error
		lastCommand, err = fileManager.CopyInlineFile("agent-config", agentConfig, remotePath, true, pulumi.DependsOn([]pulumi.Resource{lastCommand}))
		if err != nil {
			return nil, "", err
		}

	}
	return lastCommand, utils.StrHash(agentConfig), nil
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
			namer.ResourceName(confPath, utils.StrHash(content)),
			pulumi.String(content),
			confPath, true, utils.PulumiDependsOn(lastCommand))
		if err != nil {
			return nil, "", err
		}
		parts = append(parts, folderName, content)
	}

	return lastCommand, utils.StrHash(parts...), nil
}
