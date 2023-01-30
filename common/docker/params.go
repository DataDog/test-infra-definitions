package docker

import (
	"github.com/DataDog/test-infra-definitions/common"
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Params struct {
	optionalDockerAgentParams *DockerAgentParams
	composeEnvVars            map[string]string
	composeContent            string
	pulumiResources           []pulumi.ResourceOption
	commonEnv                 *config.CommonEnvironment
}

func newParams(commonEnv *config.CommonEnvironment, options ...func(*Params) error) (*Params, error) {
	return common.ApplyOption(&Params{commonEnv: commonEnv}, options)
}

func WithComposeContent(content string, env map[string]string) func(*Params) error {
	return func(p *Params) error {
		p.composeContent = content
		p.composeEnvVars = env
		return nil
	}
}

func WithPulumiResources(pulumiResources ...pulumi.ResourceOption) func(*Params) error {
	return func(p *Params) error {
		p.pulumiResources = pulumiResources
		return nil
	}
}

func WithAgent(options ...func(*DockerAgentParams) error) func(*Params) error {
	return func(p *Params) error {
		var err error
		p.optionalDockerAgentParams, err = newDockerAgentParams(p.commonEnv, options...)
		return err
	}
}
