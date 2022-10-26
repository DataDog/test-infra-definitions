package agent

import (
	"github.com/DataDog/test-infra-definitions/command"
	"github.com/DataDog/test-infra-definitions/common/config"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	agentInstallCommand = `DD_AGENT_MAJOR_VERSION=7 DD_API_KEY=%s bash -c "$(curl -L https://s3.amazonaws.com/dd-agent/scripts/install_script.sh)"`
)

func NewHostInstallation(e config.CommonEnvironment, name string, conn remote.ConnectionOutput) (*remote.Command, error) {
	return remote.NewCommand(e.Ctx, command.UniqueCommandName(name, agentInstallCommand, "", ""),
		&remote.CommandArgs{
			Connection: conn,
			Create:     pulumi.Sprintf(agentInstallCommand, e.AgentAPIKey()),
		})
}
