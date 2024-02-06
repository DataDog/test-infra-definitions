package agent

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"

	"github.com/Masterminds/semver"
)

const (
	defaultAgentImageRepo        = "gcr.io/datadoghq/agent"
	defaultClusterAgentImageRepo = "gcr.io/datadoghq/cluster-agent"
	defaultAgentImageTag         = "latest"
)

func dockerAgentFullImagePath(e *config.CommonEnvironment, repositoryPath, imageTag string) string {
	// return agent image path if defined
	if e.AgentFullImagePath() != "" {
		return e.AgentFullImagePath()
	}

	// if agent pipeline id and commit sha are defined, use the image from the pipeline pushed on agent QA registry
	if e.PipelineID() != "" && e.CommitSHA() != "" {
		return utils.BuildDockerImagePath("669783387624.dkr.ecr.us-east-1.amazonaws.com/agent", fmt.Sprintf("%s-%s", e.PipelineID(), e.CommitSHA()))
	}

	if repositoryPath == "" {
		repositoryPath = defaultAgentImageRepo
	}
	if imageTag == "" {
		imageTag = dockerAgentImageTag(e, config.AgentSemverVersion)
	}

	return utils.BuildDockerImagePath(repositoryPath, imageTag)
}

func dockerClusterAgentFullImagePath(e *config.CommonEnvironment, repositoryPath string) string {
	// return cluster agent image path if defined
	if e.ClusterAgentFullImagePath() != "" {
		return e.ClusterAgentFullImagePath()
	}

	// if agent pipeline id and commit sha are defined, use the image from the pipeline pushed on agent QA registry
	if e.PipelineID() != "" && e.CommitSHA() != "" {
		return utils.BuildDockerImagePath("669783387624.dkr.ecr.us-east-1.amazonaws.com/cluster-agent", fmt.Sprintf("%s-%s", e.PipelineID(), e.CommitSHA()))
	}

	if repositoryPath == "" {
		repositoryPath = defaultClusterAgentImageRepo
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
