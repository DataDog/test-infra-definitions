package agentinstall

import (
	"fmt"
	"strings"

	"github.com/DataDog/test-infra-definitions/aws/scenarios/vm/common"
)

type Params struct {
	apiKey      string
	version     version
	agentConfig string
}

func NewParams(apiKey string, options ...func(*Params) error) (*Params, error) {
	params := &Params{
		apiKey: apiKey,
	}
	return common.ApplyOption(params, WithLatest(), options)
}

// WithLatest uses the latest Agent 7 version in the stable channel.
func WithLatest() func(*Params) error {
	return func(p *Params) error {
		p.version.major = "7"
		p.version.betaChannel = false
		return nil
	}
}

// WithVersion use a specific version of the Agent. For example: `6.39.0` or `7.41.0~rc.7-1`
func WithVersion(version string) func(*Params) error {
	return func(p *Params) error {
		prefix := "7."
		if strings.HasPrefix(version, prefix) {
			p.version.major = "7"
		} else {
			prefix = "6."
			if strings.HasPrefix(version, prefix) {
				p.version.major = "6"
			} else {
				return fmt.Errorf("invalid version of the Agent: %v. The Agent version should starts with `7.` or `6.`", version)
			}
		}

		p.version.minor = strings.TrimPrefix(version, prefix)
		p.version.betaChannel = strings.Contains(version, "~")
		return nil
	}
}

// WithAgentConfig sets the configuration of the Agent. The configuration must contain `%v“ to set the API key.
// Example:
// `api_key: %v
// log_level: debug`
func WithAgentConfig(config string) func(*Params) error {
	return func(p *Params) error {
		p.agentConfig = config
		return nil
	}
}

type version struct {
	major       string
	minor       string
	betaChannel bool
}
