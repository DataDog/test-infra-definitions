package ecsagentparams

import "github.com/DataDog/test-infra-definitions/common"

// Params defines the parameters for the ECS Agent installation.
// The Params configuration uses the [Functional options pattern].
//
// The available options are:
//   - [WithAgentServiceEnvVariable]
//
// [Functional options pattern]: https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis

type Params struct {
	// AgentServiceEnvironment is a map of environment variables to set in the docker compose agent service's environment.
	AgentServiceEnvironment map[string]string
}

type Option = func(*Params) error

func NewParams(options ...Option) (*Params, error) {
	version := &Params{
		AgentServiceEnvironment: make(map[string]string),
	}
	return common.ApplyOption(version, options)
}

// WithAgentServiceEnvVariable set an environment variable in the docker compose agent service's environment.
func WithAgentServiceEnvVariable(key string, value string) func(*Params) error {
	return func(p *Params) error {
		p.AgentServiceEnvironment[key] = value
		return nil
	}
}
