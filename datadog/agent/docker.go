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
    image: gcr.io/datadoghq/agent:%s
    volumes:
      - "/var/run/docker.sock:/var/run/docker.sock"
      - "/proc/:/host/proc"
      - "/sys/fs/cgroup/:/host/sys/fs/cgroup"
    environment:
      DD_API_KEY: %s
      DD_PROCESS_AGENT_ENABLED: true
      DD_DOGSTATSD_NON_LOCAL_TRAFFIC: true`
)

func DockerImageTag(e config.CommonEnvironment) string {
	agentImageTag := "latest"
	agentVersion, err := config.AgentSemverVersion(e)
	if agentVersion != nil && err == nil {
		agentImageTag = agentVersion.String()
	} else {
		e.Ctx.Log.Info("Unable to parse Agent version, using latest", nil)
	}

	return agentImageTag
}

func NewDockerInstallation(e config.CommonEnvironment, dockerManager *command.DockerManager) (*remote.Command, error) {
	return dockerManager.ComposeStrUp("agent", pulumi.Sprintf(agentComposeDefinition, DockerImageTag(e), e.AgentAPIKey()))
}
