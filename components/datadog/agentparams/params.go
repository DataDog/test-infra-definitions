package agentparams

import (
	"fmt"
	"strings"

	"github.com/DataDog/test-infra-definitions/common"
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Params defines the parameters for the Agent installation.
// The Params configuration uses the [Functional options pattern].
//
// The available options are:
//   - [WithLatest]
//   - [WithVersion]
//   - [WithPipelineID]
//   - [WithRepository]
//   - [WithChannel]
//   - [WithAgentConfig]
//   - [WithIntegration]
//   - [WithTelemetry]
//   - [WithFakeintake]
//   - [WithLogs]
//
// [Functional options pattern]: https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis
type Params struct {
	Version          os.AgentVersion
	AgentConfig      string
	Integrations     map[string]string
	ExtraAgentConfig []pulumi.StringInput
}

type Option = func(*Params) error

func NewParams(env *config.CommonEnvironment, options ...Option) (*Params, error) {
	p := &Params{
		Integrations: make(map[string]string),
	}
	defaultVersion := WithLatest()
	if env.AgentVersion() != "" {
		defaultVersion = WithVersion(env.AgentVersion())
	}
	versionOptions := []Option{defaultVersion}

	// If repository and/or channel are specified, force-set them
	if env.AgentRepository() != "" {
		versionOptions = append(versionOptions, WithRepository(env.AgentRepository()))
	}
	if env.AgentChannel() != "" {
		versionOptions = append(versionOptions, WithChannel(env.AgentChannel()))
	}

	// If pipeline ID is specified, force-set parameters to testing repositories
	if env.AgentPipelineID() != "" {
		versionOptions = append(versionOptions, WithPipelineID(env.AgentPipelineID()))
	}

	options = append(versionOptions, options...)
	return common.ApplyOption(p, options)
}

// WithLatest uses the latest Agent 7 version in the stable channel.
func WithLatest() func(*Params) error {
	return func(p *Params) error {
		p.version = os.LatestAgentVersion()
		return nil
	}
}

// WithVersion use a specific version of the Agent. For example: `6.39.0` or `7.41.0~rc.7-1`
func WithVersion(version string) func(*Params) error {
	return func(p *Params) error {
		v, err := parseVersion(version)

		if err != nil {
			return err
		}
		p.Version = v

		return nil
	}
}

// WithPipelineID uses a specific testing pipeline ID of the datadog-agent CI. For example: `16497585`
func WithPipelineID(pipelineID string) func(*Params) error {
	return func(p *Params) error {
		p.version.Repository = os.TestingRepository
		p.version.PipelineID = pipelineID
		return nil
	}
}

// WithRepository uses a specific repository of the Agent. For example: `staging` or `trial`
func WithRepository(repository string) func(*Params) error {
	return func(p *Params) error {
		p.version.Repository = os.Repository(repository)
		return nil
	}
}

// WithChannel uses a specific channel of the Agent repositories. For example: `beta` or `nightly`
func WithChannel(channel string) func(*Params) error {
	return func(p *Params) error {
		p.version.Channel = os.Channel(channel)
		return nil
	}
}

func parseVersion(s string) (os.AgentVersion, error) {
	version := os.LatestAgentVersion()
	prefix := "7."
	if strings.HasPrefix(s, prefix) {
		version.Major = "7"
	} else {
		prefix = "6."
		if strings.HasPrefix(s, prefix) {
			version.Major = "6"
		} else {
			return version, fmt.Errorf("invalid version of the Agent: %v. The Agent version should starts with `7.` or `6.`", s)
		}
	}

	// Best-effort attempt to detect betas / RCs, and redirect them to the staging/beta location.
	version.Minor = strings.TrimPrefix(s, prefix)
	if strings.Contains(s, "~") {
		version.Repository = os.StagingRepository
		version.Channel = os.BetaChannel
	}
	return version, nil
}

// WithAgentConfig sets the configuration of the Agent. `{{API_KEY}}` can be used as a placeholder for the API key.
func WithAgentConfig(config string) func(*Params) error {
	return func(p *Params) error {
		p.AgentConfig = config
		return nil
	}
}

// WithIntegration adds the configuration for an integration.
func WithIntegration(folderName string, content string) func(*Params) error {
	return func(p *Params) error {
		p.Integrations[folderName] = content
		return nil
	}
}

// WithTelemetry enables the Agent telemetry go_expvar and openmetrics.
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
		p.ExtraAgentConfig = append(p.ExtraAgentConfig, pulumi.String("telemetry.enabled: true"))
		return WithIntegration("openmetrics.d", config)(p)
	}
}

// WithFakeintake installs the fake intake and configures the Agent to use it.
func WithFakeintake(fakeintake *fakeintake.ConnectionExporter) func(*Params) error {
	return func(p *Params) error {
		// configure metrics and check run intake
		extraConfig := pulumi.Sprintf(`dd_url: http://%s:80
logs_config.logs_dd_url: %s:80
logs_config.logs_no_ssl: true
logs_config.force_use_http: true`, fakeintake.Host, fakeintake.Host)
		p.ExtraAgentConfig = append(p.ExtraAgentConfig, extraConfig)
		return nil
	}
}

// WithLogs enables the log agent
func WithLogs() func(*Params) error {
	return func(p *Params) error {
		p.ExtraAgentConfig = append(p.ExtraAgentConfig, pulumi.String("logs_enabled: true"))
		return nil
	}
}
