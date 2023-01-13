package agent

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/common/config"
)

const (
	AgentComposeDefinition = `version: "3.9"
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
	DefaultAgentImageRepo = "gcr.io/datadoghq/agent"
	defaultAgentImageTag  = "latest"
)

func DockerFullImagePath(e *config.CommonEnvironment) string {
	// return agent image path if defined
	if e.AgentFullImagePath() != "" {
		return e.AgentFullImagePath()
	}

	return BuildDockerImagePath(DefaultAgentImageRepo, DockerImageTag(e))
}

func BuildDockerImagePath(dockerRepository string, imageVersion string) string {
	return fmt.Sprintf("%s:%s", dockerRepository, imageVersion)
}

func DockerImageTag(e *config.CommonEnvironment) string {
	// default tag
	agentImageTag := defaultAgentImageTag

	// try parse agent version
	agentVersion, err := config.AgentSemverVersion(e)
	if agentVersion != nil && err == nil {
		agentImageTag = agentVersion.String()
	} else {
		e.Ctx.Log.Debug("Unable to parse Agent version, using latest", nil)
	}

	return agentImageTag
}
