package dockeragentparams

import (
	"github.com/DataDog/test-infra-definitions/common"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Params defines the parameters for the Docker Agent installation.
// The Params configuration uses the [Functional options pattern].
//
// The available options are:
//   - [WithImageTag]
//   - [WithDockerRepository]
//   - [WithPulumiDependsOn]
//   - [WithEnvironmentVariables]
//
// [Functional options pattern]: https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis

type Params struct {
	// ImageTag is the docker agent image tag to use.
	ImageTag string
	// Repository is the docker repository to use.
	Repository string
	// EnvironmentVariables is a map of environment variables to set with the docker-compose context
	EnvironmentVariables pulumi.StringMap
	// PulumiDependsOn is a list of resources to depend on.
	PulumiDependsOn []pulumi.ResourceOption
}

type Option = func(*Params) error

func NewParams(options ...Option) (*Params, error) {
	version := &Params{
		EnvironmentVariables: pulumi.StringMap{},
	}
	return common.ApplyOption(version, options)
}

func WithImageTag(agentImageTag string) func(*Params) error {
	return func(p *Params) error {
		p.ImageTag = agentImageTag
		return nil
	}
}

func WithRepository(repository string) func(*Params) error {
	return func(p *Params) error {
		p.Repository = repository
		return nil
	}
}

func WithPulumiDependsOn(resources ...pulumi.ResourceOption) func(*Params) error {
	return func(p *Params) error {
		p.PulumiDependsOn = resources
		return nil
	}
}

func WithEnvironmentVariables(environmentVariables pulumi.StringMap) func(*Params) error {
	return func(p *Params) error {
		p.EnvironmentVariables = environmentVariables
		return nil
	}
}
