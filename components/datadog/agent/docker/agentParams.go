package docker

import (
	"github.com/DataDog/test-infra-definitions/common"
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/datadog/agent"
)

type AgentParams struct {
	fullImagePath string
	env           map[string]string
}

func newAgentParams(commonEnv *config.CommonEnvironment, options ...func(*AgentParams) error) (*AgentParams, error) {
	version := &AgentParams{}
	version.fullImagePath = agent.DockerFullImagePath(commonEnv, "")
	return common.ApplyOption(version, options)
}

func WithAgentImageTag(agentImageTag string) func(*AgentParams) error {
	return func(p *AgentParams) error {
		p.fullImagePath = utils.BuildDockerImagePath(agent.DefaultAgentImageRepo, agentImageTag)
		return nil
	}
}

func WithDockerAgentRepository(dockerAgentRepository string, agentImageTag string) func(*AgentParams) error {
	return func(p *AgentParams) error {
		p.fullImagePath = utils.BuildDockerImagePath(dockerAgentRepository, agentImageTag)
		return nil
	}
}

func WithEnv(env map[string]string) func(*AgentParams) error {
	return func(p *AgentParams) error {
		p.env = env
		return nil
	}
}
