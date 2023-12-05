package agent

import (
	"github.com/Masterminds/semver"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
)

const (
	DefaultAgentImageRepo        = "gcr.io/datadoghq/agent"
	DefaultClusterAgentImageRepo = "gcr.io/datadoghq/cluster-agent"
	DefaultAgentContainerName    = "datadog-agent"
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
