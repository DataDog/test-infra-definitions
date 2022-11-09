package agent

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/command"
	"github.com/DataDog/test-infra-definitions/common/config"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	agentComposeDefinition = `version: "3.9"
services:
  agent:
    image: %s
    volumes:
      - "/var/run/docker.sock:/var/run/docker.sock"
      - "/proc/:/host/proc"
      - "/sys/fs/cgroup/:/host/sys/fs/cgroup"
    environment:
      DD_API_KEY: %s
      DD_PROCESS_AGENT_ENABLED: true
      DD_DOGSTATSD_NON_LOCAL_TRAFFIC: true`
)

func DockerImage(e config.CommonEnvironment) string {
	// return agent image path if defined
	if e.AgentImagePath() != "" {
		return e.AgentImagePath()
	}

	// default repo
	agentImage := "gcr.io/datadoghq/agent"
	// default tag
	agentImageTag := "latest"

	// try parse agent version
	agentVersion, err := config.AgentSemverVersion(e)
	if err == nil {
		agentImageTag = agentVersion.String()
	}
	e.Ctx.Log.Debug("Unable to parse Agent version, using latest", nil)

	return fmt.Sprintf("%s:%s", agentImage, agentImageTag)
}

func NewDockerInstallation(e config.CommonEnvironment, dockerManager *command.DockerManager) (*remote.Command, error) {
	return dockerManager.ComposeStrUp("agent", pulumi.Sprintf(agentComposeDefinition, DockerImage(e), e.AgentAPIKey()))
}
