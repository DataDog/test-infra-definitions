package agent

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/command"
	"github.com/DataDog/test-infra-definitions/common/config"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	agentInstallCommand = `DD_API_KEY=%s %s bash -c "$(curl -L https://s3.amazonaws.com/dd-agent/scripts/install_script.sh)"`
)

func NewHostInstallation(e config.CommonEnvironment, name string, conn remote.ConnectionOutput) (*remote.Command, error) {
	var extraParameters string
	agentVersion, err := config.AgentSemverVersion(&e)
	if err != nil {
		e.Ctx.Log.Info("Unable to parse Agent version, using latest", nil)
	}

	if agentVersion == nil {
		extraParameters += "DD_AGENT_MAJOR_VERSION=7"
	} else {
		extraParameters += fmt.Sprintf("DD_AGENT_MAJOR_VERSION=%d DD_AGENT_MINOR_VERSION=%d", agentVersion.Major(), agentVersion.Minor())
	}

	return remote.NewCommand(e.Ctx, command.UniqueCommandName(name, agentInstallCommand, "", ""),
		&remote.CommandArgs{
			Connection: conn,
			Create:     pulumi.Sprintf(agentInstallCommand, e.AgentAPIKey(), extraParameters),
		})
}
