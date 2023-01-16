package agentinstall

import (
	"fmt"
	"strings"

	"github.com/DataDog/test-infra-definitions/command"
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/os"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Install(runner *command.Runner, env *config.CommonEnvironment, params *Params, os os.OS) (pulumi.Resource, error) {
	cmd := getInstallFormatString(os.GetOSType(), params.version)
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

func getInstallFormatString(osType os.OSType, version version) string {
	switch osType {
	case os.UbuntuOS:
		return getUnixInstallFormatString("install_script.sh", version)
	case os.MacosOS:
		return getUnixInstallFormatString("install_mac_os.sh", version)
	case os.WindowsOS:
		panic("Not implemented")
	default:
		panic("Not implemented")
	}
}

func getUnixInstallFormatString(scriptName string, version version) string {
	commandLine := fmt.Sprintf("DD_AGENT_MAJOR_VERSION=%v ", version.major)

	if version.minor != "" {
		commandLine += fmt.Sprintf("DD_AGENT_MINOR_VERSION=%v ", version.minor)
	}

	if version.betaChannel {
		commandLine += "REPO_URL=datad0g.com DD_AGENT_DIST_CHANNEL=beta "
	}

	return fmt.Sprintf(
		`DD_API_KEY=%%s %v DD_INSTALL_ONLY=true bash -c "$(curl -L https://s3.amazonaws.com/dd-agent/scripts/%v)"`,
		commandLine,
		scriptName)
}
