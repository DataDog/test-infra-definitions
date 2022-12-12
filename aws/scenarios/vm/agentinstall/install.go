package agentinstall

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/aws/scenarios/vm/os"
	"github.com/DataDog/test-infra-definitions/command"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Install(runner *command.Runner, env aws.Environment, params *Params, os os.OS) error {
	cmd := getInstallFormatString(os.GetOSType(), params.version)
	lastCommand, err := runner.Command(
		env.CommonNamer.ResourceName("agent-install", utils.StrHash(cmd)),
		&command.CommandArgs{
			Create: pulumi.Sprintf(cmd, params.apiKey),
		})
	if err != nil {
		return err
	}

	agentConfig := ""
	if params.agentConfig != "" {
		fileManager := command.NewFileManager(runner)
		agentConfig = fmt.Sprintf(params.agentConfig, params.apiKey)
		remotePath := os.GetConfigPath()
		lastCommand, err = fileManager.CopyInlineFile("agent-config", agentConfig, remotePath, true, pulumi.DependsOn([]pulumi.Resource{lastCommand}))
		if err != nil {
			return err
		}

	}

	// When the file content has changed, make sure the Agent is restarted.
	serviceManager := os.GetServiceManager()
	startAgentRes := env.CommonNamer.ResourceName("start-agent", utils.StrHash(serviceManager.StartAgentCmd(), agentConfig))
	_, err = runner.Command(
		startAgentRes,
		&command.CommandArgs{
			Create: pulumi.String(serviceManager.StartAgentCmd()),
		}, pulumi.DependsOn([]pulumi.Resource{lastCommand}))

	return err
}

func getInstallFormatString(osType os.OSType, version version) string {
	switch osType {
	case os.UbuntuOS:
		return getUnixInstallFormatString("install_script.sh", version)
	case os.MacOS:
		return getUnixInstallFormatString("install_mac_os.sh", version)
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
