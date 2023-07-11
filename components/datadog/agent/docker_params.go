package agent

import (
	"github.com/DataDog/test-infra-definitions/common"
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
)

type dockerParams struct {
	env config.CommonEnvironment

	fullImagePath  string
	composeEnvVars map[string]string
	composeContent string
}

type DockerOption = func(*dockerParams) error

func newDockerParams(e config.CommonEnvironment, options ...DockerOption) (*dockerParams, error) {
	return common.ApplyOption(&dockerParams{}, options)
}

func WithComposeContent(content string, envVars map[string]string) DockerOption {
	return func(p *dockerParams) error {
		p.composeContent = content
		p.composeEnvVars = envVars
		return nil
	}
}

func WithAgentFullImagePath(fullImagePath string) DockerOption {
	return func(p *dockerParams) error {
		p.fullImagePath = fullImagePath
		return nil
	}
}

func WithAgentImageTag(agentImageTag string) DockerOption {
	return func(p *dockerParams) error {
		p.fullImagePath = utils.BuildDockerImagePath(defaultAgentImageRepo, agentImageTag)
		return nil
	}
}

func WithDockerAgentRepository(dockerAgentRepository string, agentImageTag string) DockerOption {
	return func(p *dockerParams) error {
		p.fullImagePath = utils.BuildDockerImagePath(dockerAgentRepository, agentImageTag)
		return nil
	}
}
