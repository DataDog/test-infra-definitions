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
	defaultOTAgentImageRepo      = "datadog/agent-dev"
	defaultOTAgentImageTag       = "nightly-ot-beta-main"
)

func dockerAgentFullImagePath(e config.Env, repositoryPath, imageTag string, otel bool, useLatestStableAgent bool) string {
	if !useLatestStableAgent {
		// return agent image path if defined
		if e.AgentFullImagePath() != "" {
			return e.AgentFullImagePath()
		}

		// if agent pipeline id and commit sha are defined, use the image from the pipeline pushed on agent QA registry
		if e.PipelineID() != "" && e.CommitSHA() != "" && imageTag == "" {
			tag := fmt.Sprintf("%s-%s", e.PipelineID(), e.CommitSHA())
			if otel {
				tag = fmt.Sprintf("%s-7-ot-beta", tag)
			}

			exists, err := e.InternalRegistryImageTagExists(fmt.Sprintf("%s/agent", e.InternalRegistry()), tag)
			if err != nil || !exists {
				panic(fmt.Sprintf("image %s/agent:%s not found in the internal registry", e.InternalRegistry(), tag))
			}
			return utils.BuildDockerImagePath(fmt.Sprintf("%s/agent", e.InternalRegistry()), tag)
		}
	}

	if repositoryPath == "" && otel {
		repositoryPath = defaultOTAgentImageRepo
	}

	if repositoryPath == "" {
		repositoryPath = defaultAgentImageRepo
	}

	if imageTag == "" && otel {
		imageTag = defaultOTAgentImageTag
	}

	if imageTag == "" {
		imageTag = dockerAgentImageTag(e, config.AgentSemverVersion)
	}

	return utils.BuildDockerImagePath(repositoryPath, imageTag)
}

func dockerClusterAgentFullImagePath(e config.Env, repositoryPath string) string {
	// return cluster agent image path if defined
	if e.ClusterAgentFullImagePath() != "" {
		return e.ClusterAgentFullImagePath()
	}

	// if agent pipeline id and commit sha are defined, use the image from the pipeline pushed on agent QA registry
	if e.PipelineID() != "" && e.CommitSHA() != "" {
		tag := fmt.Sprintf("%s-%s", e.PipelineID(), e.CommitSHA())

		exists, err := e.InternalRegistryImageTagExists(fmt.Sprintf("%s/cluster-agent", e.InternalRegistry()), tag)
		if err != nil || !exists {
			panic(fmt.Sprintf("image %s/cluster-agent:%s not found in the internal registry", e.InternalRegistry(), tag))
		}
		return utils.BuildDockerImagePath(fmt.Sprintf("%s/cluster-agent", e.InternalRegistry()), tag)
	}

	if repositoryPath == "" {
		repositoryPath = defaultClusterAgentImageRepo
	}

	return utils.BuildDockerImagePath(repositoryPath, dockerAgentImageTag(e, config.ClusterAgentSemverVersion))
}

func dockerAgentImageTag(e config.Env, semverVersion func(config.Env) (*semver.Version, error)) string {
	// default tag
	agentImageTag := defaultAgentImageTag

	// try parse agent version
	agentVersion, err := semverVersion(e)
	if agentVersion != nil && err == nil {
		agentImageTag = agentVersion.String()
	} else {
		e.Ctx().Log.Debug("Unable to parse agent version, using latest", nil)
	}

	return agentImageTag
}
