package agentparams

import (
	"fmt"
	"path"
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
//   - [WithPipeline]
//   - [WithAgentConfig]
//   - [WithSystemProbeConfig]
//   - [WithSecurityAgentConfig]
//   - [WithFile]
//   - [WithIntegration]
//   - [WithTelemetry]
//   - [WithFakeintake]
//   - [WithIntakeHostname]
//   - [WithLogs]
//
// [Functional options pattern]: https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis

type FileDefinition struct {
	Content string
	UseSudo bool
}

type Params struct {
	Version             os.AgentVersion
	AgentConfig         string
	SystemProbeConfig   string
	SecurityAgentConfig string
	Integrations        map[string]*FileDefinition
	Files               map[string]*FileDefinition
	ExtraAgentConfig    []pulumi.StringInput
}

type Option = func(*Params) error

func NewParams(env *config.CommonEnvironment, options ...Option) (*Params, error) {
	p := &Params{
		Integrations: make(map[string]*FileDefinition),
		Files:        make(map[string]*FileDefinition),
	}
	defaultVersion := WithLatest()
	if env.PipelineID() != "" {
		defaultVersion = WithPipeline(env.PipelineID(), env.InfraOSArchitecture())
	}
	if env.AgentVersion() != "" {
		defaultVersion = WithVersion(env.AgentVersion())
	}
	options = append([]Option{defaultVersion}, options...)
	return common.ApplyOption(p, options)
}

// WithLatest uses the latest Agent 7 version in the stable channel.
func WithLatest() func(*Params) error {
	return func(p *Params) error {
		p.Version.Major = "7"
		p.Version.BetaChannel = false
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

// WithPipeline use a specific version of the Agent by pipeline id. For example: `16497585` uses the version `pipeline-16497585`
func WithPipeline(pipelineID string, arch string) func(*Params) error {
	return func(p *Params) error {
		p.Version = parsePipelineVersion(pipelineID, arch)
		return nil
	}
}

func parseVersion(s string) (os.AgentVersion, error) {
	version := os.AgentVersion{}

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
	version.Minor = strings.TrimPrefix(s, prefix)
	version.BetaChannel = strings.Contains(s, "~")
	return version, nil
}

func parsePipelineVersion(s string, arch string) os.AgentVersion {
	version := os.AgentVersion{}
	version.PipelineID = "pipeline-" + s
	version.Arch = arch
	return version
}

// WithAgentConfig sets the configuration of the Agent. `{{API_KEY}}` can be used as a placeholder for the API key.
func WithAgentConfig(config string) func(*Params) error {
	return func(p *Params) error {
		p.AgentConfig = config
		return nil
	}
}

// WithSystemProbeConfig sets the configuration of system-probe.
func WithSystemProbeConfig(config string) func(*Params) error {
	return func(p *Params) error {
		p.SystemProbeConfig = config
		return nil
	}
}

// WithSecurityAgentConfig sets the configuration of the security-agent.
func WithSecurityAgentConfig(config string) func(*Params) error {
	return func(p *Params) error {
		p.SecurityAgentConfig = config
		return nil
	}
}

// WithIntegration adds the configuration for an integration.
func WithIntegration(folderName string, content string) func(*Params) error {
	return func(p *Params) error {
		confPath := path.Join("conf.d", folderName, "conf.yaml")
		p.Integrations[confPath] = &FileDefinition{
			Content: content,
			UseSudo: true,
		}
		return nil
	}
}

// WithFile adds a file with contents to the install at the given path. This should only be used when the agent needs to be restarted after writing the file.
func WithFile(absolutePath string, content string, useSudo bool) func(*Params) error {
	return func(p *Params) error {
		p.Files[absolutePath] = &FileDefinition{
			Content: content,
			UseSudo: useSudo,
		}
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

func withIntakeHostname(hostname pulumi.StringInput) func(*Params) error {
	return func(p *Params) error {
		extraConfig := pulumi.Sprintf(`dd_url: http://%[1]s:80
logs_config.logs_dd_url: %[1]s:80
logs_config.logs_no_ssl: true
logs_config.force_use_http: true
process_config.process_dd_url: http://%[1]s:80
database_monitoring.metrics.logs_dd_url: %[1]s:80
database_monitoring.metrics.logs_no_ssl: true
database_monitoring.activity.logs_dd_url: %[1]s:80
database_monitoring.activity.logs_no_ssl: true
database_monitoring.samples.logs_dd_url: %[1]s:80
database_monitoring.samples.logs_no_ssl: true
network_devices.metadata.logs_dd_url: %[1]s:80
network_devices.metadata.logs_no_ssl: true
network_devices.snmp_traps.forwarder.logs_dd_url: %[1]s:80
network_devices.snmp_traps.forwarder.logs_no_ssl: true
network_devices.netflow.forwarder.logs_dd_url: %[1]s:80
network_devices.netflow.forwarder.logs_no_ssl: true
container_lifecycle.logs_dd_url: %[1]s:80
container_lifecycle.logs_no_ssl: true
container_image.logs_dd_url: %[1]s:80
container_image.logs_no_ssl: true
sbom.logs_dd_url: %[1]s:80
sbom.logs_no_ssl: true`, hostname)
		p.ExtraAgentConfig = append(p.ExtraAgentConfig, extraConfig)
		return nil
	}
}

// WithIntakeName configures the agent to use the given hostname as intake.
//
// To use a fakeintake, see WithFakeintake.
//
// This option is overwritten by `WithFakeintake`.
func WithIntakeHostname(hostname string) func(*Params) error {
	return withIntakeHostname(pulumi.String(hostname))
}

// WithFakeintake installs the fake intake and configures the Agent to use it.
//
// This option is overwritten by `WithIntakeHostname`.
func WithFakeintake(fakeintake *fakeintake.ConnectionExporter) func(*Params) error {
	return withIntakeHostname(fakeintake.Host)
}

// WithLogs enables the log agent
func WithLogs() func(*Params) error {
	return func(p *Params) error {
		p.ExtraAgentConfig = append(p.ExtraAgentConfig, pulumi.String("logs_enabled: true"))
		return nil
	}
}
