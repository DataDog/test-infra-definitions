package dockeragentparams

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/DataDog/test-infra-definitions/common"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Params defines the parameters for the Docker Agent installation.
// The Params configuration uses the [Functional options pattern].
//
// The available options are:
//   - [WithImageTag]
//   - [WithRepository]
//   - [WithPulumiDependsOn]
//   - [WithEnvironmentVariables]
//   - [WithAgentServiceEnvVariable]
//   - [WithHostName]
//	 - [WithFakeintake]
//	 - [WithLogs]
//   - [WithExtraComposeManifest]
//
// [Functional options pattern]: https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis

type Params struct {
	// ImageTag is the docker agent image tag to use.
	ImageTag string
	// Repository is the docker repository to use.
	Repository string
	// AgentServiceEnvironment is a map of environment variables to set in the docker compose agent service's environment.
	AgentServiceEnvironment pulumi.Map
	// ExtraComposeManifests is a list of extra docker compose manifests to add beside the agent service.
	ExtraComposeManifests []command.DockerComposeInlineManifest
	// EnvironmentVariables is a map of environment variables to set with the docker-compose context
	EnvironmentVariables pulumi.StringMap
	// PulumiDependsOn is a list of resources to depend on.
	PulumiDependsOn []pulumi.ResourceOption
}

type Option = func(*Params) error

func NewParams(options ...Option) (*Params, error) {
	version := &Params{
		AgentServiceEnvironment: pulumi.Map{},
		EnvironmentVariables:    pulumi.StringMap{},
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

// WithAgentServiceEnvVariable set an environment variable in the docker compose agent service's environment.
func WithAgentServiceEnvVariable(key string, value pulumi.Input) func(*Params) error {
	return func(p *Params) error {
		p.AgentServiceEnvironment[key] = value
		return nil
	}
}

// WithIntake configures the agent to use the given hostname as intake.
// The hostname must be a valid Datadog intake, with a SSL valid certificate
//
// To use a fakeintake, see WithFakeintake.
//
// This option is overwritten by `WithFakeintake`.
func WithIntake(hostname string) func(*Params) error {
	return withIntakeHostname(pulumi.String(hostname), false)
}

// WithFakeintake installs the fake intake and configures the Agent to use it.
//
// This option is overwritten by `WithIntakeHostname`.
func WithFakeintake(fakeintake *fakeintake.ConnectionExporter) func(*Params) error {
	return withIntakeHostname(fakeintake.Host, true)
}

func withIntakeHostname(hostname pulumi.StringInput, shouldSkipSSLCertificateValidation bool) func(*Params) error {
	shouldEnforceHTTPInput := hostname.ToStringOutput().ApplyT(func(host string) (bool, error) {
		return strings.HasPrefix(host, "https"), nil
	}).(pulumi.BoolOutput)
	return func(p *Params) error {
		logsEnvVars := pulumi.Map{
			"DD_DD_URL":                        pulumi.Sprintf("http://%s:80", hostname),
			"DD_LOGS_CONFIG_DD_URL":            pulumi.Sprintf("%s:80", hostname),
			"DD_PROCESS_CONFIG_PROCESS_DD_URL": pulumi.Sprintf("http://%s:80", hostname),
			"DD_SKIP_SSL_VALIDATION":           pulumi.Bool(shouldSkipSSLCertificateValidation),
			"DD_LOGS_CONFIG_LOGS_NO_SSL":       pulumi.Bool(shouldSkipSSLCertificateValidation),
			"DD_LOGS_CONFIG_FORCE_USE_HTTP":    shouldEnforceHTTPInput,
		}
		for key, value := range logsEnvVars {
			if err := WithAgentServiceEnvVariable(key, value)(p); err != nil {
				return err
			}
		}
		return nil
	}
}

type additionalLogEndpointInput struct {
	Hostname   string `json:"host"`
	APIKey     string `json:"api_key,omitempty"`
	Port       string `json:"port,omitempty"`
	IsReliable bool   `json:"is_reliable,omitempty"`
}

func WithAdditionalFakeintake(fakeintake *fakeintake.ConnectionExporter) func(*Params) error {
	additionalEndpointsContentInput := fakeintake.Host.ToStringOutput().ApplyT(func(host string) (string, error) {
		endpoints := map[string][]string{
			fmt.Sprintf("https://%s", host): {"00000000000000000000000000000000"},
		}

		jsonContent, err := json.Marshal(endpoints)
		return string(jsonContent), err
	}).(pulumi.StringOutput)
	additionalLogsEndpointsContentInput := fakeintake.Host.ToStringOutput().ApplyT(func(host string) (string, error) {
		endpoints := []additionalLogEndpointInput{
			{
				Hostname: host,
			},
		}
		jsonContent, err := json.Marshal(endpoints)
		return string(jsonContent), err
	}).(pulumi.StringOutput)
	shouldEnforceHTTPInput := fakeintake.Host.ToStringOutput().ApplyT(func(host string) (bool, error) {
		return strings.HasPrefix(host, "https"), nil
	}).(pulumi.BoolOutput)
	return func(p *Params) error {
		logsEnvVars := pulumi.Map{
			"DD_ADDITIONAL_ENDPOINTS":             additionalEndpointsContentInput,
			"DD_LOGS_CONFIG_ADDITIONAL_ENDPOINTS": additionalLogsEndpointsContentInput,
			"DD_SKIP_SSL_VALIDATION":              pulumi.Bool(true),
			"DD_LOGS_CONFIG_LOGS_NO_SSL":          pulumi.Bool(true),
			"DD_LOGS_CONFIG_FORCE_USE_HTTP":       shouldEnforceHTTPInput,
		}
		for key, value := range logsEnvVars {
			if err := WithAgentServiceEnvVariable(key, value)(p); err != nil {
				return err
			}
		}
		return nil
	}
}

// WithLogs enables the log agent
func WithLogs() func(*Params) error {
	return WithAgentServiceEnvVariable("DD_LOGS_ENABLED", pulumi.String("true"))
}

// WithExtraComposeContent adds a cpm
func WithExtraComposeManifest(name, content string) func(*Params) error {
	return func(p *Params) error {
		p.ExtraComposeManifests = append(p.ExtraComposeManifests, command.DockerComposeInlineManifest{
			Name:    name,
			Content: pulumi.String(content),
		})
		return nil
	}
}
