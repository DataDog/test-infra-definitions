package agent

import (
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
	agentImage := "gcr.io/datadoghq/agent:latest"
	if e.AgentImagePath() != "" {
		agentImage = e.AgentImagePath()
	}

	return agentImage
}

func NewDockerInstallation(e config.CommonEnvironment, dockerManager *command.DockerManager) (*remote.Command, error) {
	return dockerManager.ComposeStrUp("agent", pulumi.Sprintf(agentComposeDefinition, DockerImage(e), e.AgentAPIKey()))
}
