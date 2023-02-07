package agentinstall

import (
	"fmt"
	"strings"

	"github.com/DataDog/test-infra-definitions/common"
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/os"
)

type params struct {
	version     os.AgentVersion
	agentConfig string
}

func newParams(env *config.CommonEnvironment, options ...func(*params) error) (*params, error) {
	p := &params{}
	defaultVersion := WithLatest()
	if env.AgentVersion() != "" {
		defaultVersion = WithVersion(env.AgentVersion())
	}
	options = append([]func(*params) error{defaultVersion}, options...)
	return common.ApplyOption(p, options)
}

// WithLatest uses the latest Agent 7 version in the stable channel.
func WithLatest() func(*params) error {
	return func(p *params) error {
		p.version.Major = "7"
		p.version.BetaChannel = false
		return nil
	}
}

// WithVersion use a specific version of the Agent. For example: `6.39.0` or `7.41.0~rc.7-1`
func WithVersion(version string) func(*params) error {
	return func(p *params) error {
		prefix := "7."
		if strings.HasPrefix(version, prefix) {
			p.version.Major = "7"
		} else {
			prefix = "6."
			if strings.HasPrefix(version, prefix) {
				p.version.Major = "6"
			} else {
				return fmt.Errorf("invalid version of the Agent: %v. The Agent version should starts with `7.` or `6.`", version)
			}
		}

		p.version.Minor = strings.TrimPrefix(version, prefix)
		p.version.BetaChannel = strings.Contains(version, "~")
		return nil
	}
}

// WithAgentConfig sets the configuration of the Agent. `{{API_KEY}}` can be used as a placeholder for the API key.
func WithAgentConfig(config string) func(*params) error {
	return func(p *params) error {
		p.agentConfig = config
		return nil
	}
}
