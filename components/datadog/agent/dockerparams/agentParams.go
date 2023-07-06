package dockerparams

import (
	"github.com/DataDog/test-infra-definitions/common"
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/datadog/agent"
)

type AgentParams struct {
	FullImagePath string
	Env           map[string]string
}

type AgentOption = func(*AgentParams) error

func newAgentParams(commonEnv *config.CommonEnvironment, options ...AgentOption) (*AgentParams, error) {
	version := &AgentParams{}
	version.FullImagePath = agent.DockerAgentFullImagePath(commonEnv, "")
	return common.ApplyOption(version, options)
}

func WithAgentImageTag(agentImageTag string) func(*AgentParams) error {
	return func(p *AgentParams) error {
		p.FullImagePath = utils.BuildDockerImagePath(agent.DefaultAgentImageRepo, agentImageTag)
		return nil
	}
}

func WithDockerAgentRepository(dockerAgentRepository string, agentImageTag string) func(*AgentParams) error {
	return func(p *AgentParams) error {
		p.FullImagePath = utils.BuildDockerImagePath(dockerAgentRepository, agentImageTag)
		return nil
	}
}

func WithEnv(env map[string]string) func(*AgentParams) error {
	return func(p *AgentParams) error {
		p.Env = env
		return nil
	}
}
