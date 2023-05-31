package agent

import (
	"github.com/Masterminds/semver"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
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
	DefaultAgentImageRepo        = "gcr.io/datadoghq/agent"
	DefaultClusterAgentImageRepo = "gcr.io/datadoghq/cluster-agent"
	defaultAgentImageTag         = "latest"
)

func DockerAgentFullImagePath(e *config.CommonEnvironment, repositoryPath string) string {
	// return agent image path if defined
	if e.AgentFullImagePath() != "" {
		return e.AgentFullImagePath()
	}

	if repositoryPath == "" {
		repositoryPath = DefaultAgentImageRepo
	}

	return utils.BuildDockerImagePath(repositoryPath, dockerAgentImageTag(e, config.AgentSemverVersion))
}

func DockerClusterAgentFullImagePath(e *config.CommonEnvironment, repositoryPath string) string {
	// return cluster agent image path if defined
	if e.ClusterAgentFullImagePath() != "" {
		return e.ClusterAgentFullImagePath()
	}

	if repositoryPath == "" {
		repositoryPath = DefaultClusterAgentImageRepo
	}

	return utils.BuildDockerImagePath(repositoryPath, dockerAgentImageTag(e, config.ClusterAgentSemverVersion))
}

func dockerAgentImageTag(e *config.CommonEnvironment, semverVersion func(*config.CommonEnvironment) (*semver.Version, error)) string {
	// default tag
	agentImageTag := defaultAgentImageTag

	// try parse agent version
	agentVersion, err := semverVersion(e)
	if agentVersion != nil && err == nil {
		agentImageTag = agentVersion.String()
	} else {
		e.Ctx.Log.Debug("Unable to parse agent version, using latest", nil)
	}

	return agentImageTag
}
