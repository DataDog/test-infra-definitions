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
	agentCommand := &agentCommand{version: params.version}
	os.Visit(agentCommand)
	cmd := agentCommand.cmd
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

type agentCommand struct {
	version version
	cmd     string
}

func (a *agentCommand) VisitUnix()    { a.cmd = getInstallFormalString("install_script.sh", a.version) }
func (a *agentCommand) VisitMacOS()   { a.cmd = getInstallFormalString("install_mac_os.sh", a.version) }
func (a *agentCommand) VisitWindows() { panic("Not implemented") }

func getInstallFormalString(scriptName string, version version) string {
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
