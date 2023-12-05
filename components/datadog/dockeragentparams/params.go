package dockeragentparams

import (
	"github.com/DataDog/test-infra-definitions/common"
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Params struct {
	OptionalDockerAgentParams *AgentParams
	ComposeEnvVars            map[string]string
	ComposeContent            string
	PulumiResources           []pulumi.ResourceOption
	CommonEnv                 *config.CommonEnvironment
	Architecture              os.Architecture
}

type Option = func(*Params) error

func NewParams(commonEnv *config.CommonEnvironment, options ...Option) (*Params, error) {
	return common.ApplyOption(&Params{CommonEnv: commonEnv, Architecture: os.AMD64Arch}, options)
}

func WithComposeContent(content string, env map[string]string) func(*Params) error {
	return func(p *Params) error {
		p.ComposeContent = content
		p.ComposeEnvVars = env
		return nil
	}
}

func WithPulumiResources(pulumiResources ...pulumi.ResourceOption) func(*Params) error {
	return func(p *Params) error {
		p.PulumiResources = pulumiResources
		return nil
	}
}

func WithAgent(options ...AgentOption) func(*Params) error {
	return func(p *Params) error {
		var err error
		p.OptionalDockerAgentParams, err = newAgentParams(p.CommonEnv, options...)
		return err
	}
}

func WithArchitecture(arch os.Architecture) func(*Params) error {
	return func(p *Params) error {
		p.Architecture = arch
		return nil
	}
}
