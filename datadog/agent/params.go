package agent

import (
	"fmt"
	"strings"

	"github.com/DataDog/test-infra-definitions/common"
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/os"
)

type Params struct {
	version          os.AgentVersion
	agentConfig      string
	integrations     map[string]string
	extraAgentConfig []string
}

func newParams(env *config.CommonEnvironment, options ...func(*Params) error) (*Params, error) {
	p := &Params{
		integrations: make(map[string]string),
	}
	defaultVersion := WithLatest()
	if env.AgentVersion() != "" {
		defaultVersion = WithVersion(env.AgentVersion())
	}
	options = append([]func(*Params) error{defaultVersion}, options...)
	return common.ApplyOption(p, options)
}

// WithLatest uses the latest Agent 7 version in the stable channel.
func WithLatest() func(*Params) error {
	return func(p *Params) error {
		p.version.Major = "7"
		p.version.BetaChannel = false
		return nil
	}
}

// WithVersion use a specific version of the Agent. For example: `6.39.0` or `7.41.0~rc.7-1`
func WithVersion(version string) func(*Params) error {
	return func(p *Params) error {
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

// WithAgentConfig sets the configuration of the Agent. There is no need to include `api_key“.
func WithAgentConfig(config string) func(*Params) error {
	return func(p *Params) error {
		p.agentConfig = config
		return nil
	}
}

// WithIntegration adds the configuration for an integration.
func WithIntegration(folderName string, content string) func(*Params) error {
	return func(p *Params) error {
		p.integrations[folderName] = content
		return nil
	}
}

func WithTelemetry() func(*Params) error {
	return func(p *Params) error {
		config := `instances:
  - expvar_url: http://localhost:5000/debug/vars
    max_returned_metrics: 1000
    metrics:      
      - path: ".*"
      - path: ".*/.*"
      - path: ".*/.*/.*"
`
		if err := WithIntegration("go_expvar.d", config)(p); err != nil {
			return err
		}

		config = `instances:
  - prometheus_url: http://localhost:5000/telemetry
    namespace: "datadog"
    metrics:
      - "*"
`
		p.extraAgentConfig = append(p.extraAgentConfig, "telemetry.enabled: true")
		return WithIntegration("openmetrics.d", config)(p)
	}
}
