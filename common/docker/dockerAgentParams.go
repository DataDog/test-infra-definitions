package docker

import (
	"github.com/DataDog/test-infra-definitions/common"
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/datadog/agent"
)

type DockerAgentParams struct {
	fullImagePath string
	env           map[string]string
}

func newDockerAgentParams(commonEnv *config.CommonEnvironment, options ...func(*DockerAgentParams) error) (*DockerAgentParams, error) {
	version := &DockerAgentParams{}
	version.fullImagePath = agent.DockerFullImagePath(commonEnv)
	return common.ApplyOption(version, options)
}

func WithAgentImageTag(agentImageTag string) func(*DockerAgentParams) error {
	return func(p *DockerAgentParams) error {
		p.fullImagePath = agent.BuildDockerImagePath(agent.DefaultAgentImageRepo, agentImageTag)
		return nil
	}
}

func WithDockerAgentRepository(dockerAgentRepository string, agentImageTag string) func(*DockerAgentParams) error {
	return func(p *DockerAgentParams) error {
		p.fullImagePath = agent.BuildDockerImagePath(dockerAgentRepository, agentImageTag)
		return nil
	}
}

func WithEnv(env map[string]string) func(*DockerAgentParams) error {
	return func(p *DockerAgentParams) error {
		p.env = env
		return nil
	}
}
