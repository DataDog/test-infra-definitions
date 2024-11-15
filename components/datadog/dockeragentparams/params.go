package dockeragentparams

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/DataDog/test-infra-definitions/common"
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
	"github.com/DataDog/test-infra-definitions/components/docker"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Params defines the parameters for the Docker Agent installation.
// The Params configuration uses the [Functional options pattern].
//
// The available options are:
//   - [WithImageTag]
//   - [WithRepository]
//   - [WithFullImagePath]
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
	// FullImagePath is the full path of the docker agent image to use.
	// It has priority over ImageTag and Repository.
	FullImagePath string
	// ImageTag is the docker agent image tag to use.
	ImageTag string
	// Repository is the docker repository to use.
	Repository string
	// JMX is true if the JMX image is needed
	JMX bool
	// AgentServiceEnvironment is a map of environment variables to set in the docker compose agent service's environment.
	AgentServiceEnvironment pulumi.Map
	// ExtraComposeManifests is a list of extra docker compose manifests to add beside the agent service.
	ExtraComposeManifests []docker.ComposeInlineManifest
	// EnvironmentVariables is a map of environment variables to set with the docker-compose context
	EnvironmentVariables pulumi.StringMap
	// PulumiDependsOn is a list of resources to depend on.
	PulumiDependsOn []pulumi.ResourceOption
}

type Option = func(*Params) error

func NewParams(e config.Env, options ...Option) (*Params, error) {
	version := &Params{
		AgentServiceEnvironment: pulumi.Map{},
		EnvironmentVariables:    pulumi.StringMap{},
	}

	if e.PipelineID() != "" && e.CommitSHA() != "" {
		exists, err := e.InternalRegistryImageTagExists(fmt.Sprintf("%s/agent", e.InternalRegistry()), fmt.Sprintf("%s-%s", e.PipelineID(), e.CommitSHA()))
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, fmt.Errorf("image %s/agent:%s not found in the internal registry", e.InternalRegistry(), fmt.Sprintf("%s-%s", e.PipelineID(), e.CommitSHA()))
		}
		options = append(options, WithFullImagePath(utils.BuildDockerImagePath("669783387624.dkr.ecr.us-east-1.amazonaws.com/agent", fmt.Sprintf("%s-%s", e.PipelineID(), e.CommitSHA()))))
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

// WithJMX makes the image be the one with Java installed
func WithJMX() func(*Params) error {
	return func(p *Params) error {
		p.JMX = true
		return nil
	}
}

func WithFullImagePath(fullImagePath string) func(*Params) error {
	return func(p *Params) error {
		p.FullImagePath = fullImagePath
		return nil
	}
}

func WithPulumiDependsOn(resources ...pulumi.ResourceOption) func(*Params) error {
	return func(p *Params) error {
		p.PulumiDependsOn = append(p.PulumiDependsOn, resources...)
		return nil
	}
}

func WithEnvironmentVariables(environmentVariables pulumi.StringMap) func(*Params) error {
	return func(p *Params) error {
		p.EnvironmentVariables = environmentVariables
		return nil
	}
}

func WithTags(tags []string) func(*Params) error {
	return WithAgentServiceEnvVariable("DD_TAGS", pulumi.String(strings.Join(tags, ",")))
}

// WithAgentServiceEnvVariable set an environment variable in the docker compose agent service's environment.
func WithAgentServiceEnvVariable(key string, value pulumi.Input) func(*Params) error {
	return func(p *Params) error {
		p.AgentServiceEnvironment[key] = value
		return nil
	}
}

// WithIntake configures the agent to use the given url as intake.
// The url must be a valid Datadog intake, with a SSL valid certificate
//
// To use a fakeintake, see WithFakeintake.
//
// This option is overwritten by `WithFakeintake`.
func WithIntake(url string) func(*Params) error {
	return withIntakeHostname(pulumi.String(url), false)
}

// WithFakeintake installs the fake intake and configures the Agent to use it.
//
// This option is overwritten by `WithIntakeHostname`.
func WithFakeintake(fakeintake *fakeintake.Fakeintake) func(*Params) error {
	shouldSkipSSLValidation := fakeintake.Scheme == "http"
	return func(p *Params) error {
		p.PulumiDependsOn = append(p.PulumiDependsOn, utils.PulumiDependsOn(fakeintake))
		return withIntakeHostname(fakeintake.URL, shouldSkipSSLValidation)(p)
	}
}

func withIntakeHostname(url pulumi.StringInput, shouldSkipSSLValidation bool) func(*Params) error {
	return func(p *Params) error {
		envVars := pulumi.Map{
			"DD_DD_URL":                                  pulumi.Sprintf("%s", url),
			"DD_PROCESS_CONFIG_PROCESS_DD_URL":           pulumi.Sprintf("%s", url),
			"DD_APM_DD_URL":                              pulumi.Sprintf("%s", url),
			"DD_SKIP_SSL_VALIDATION":                     pulumi.Bool(shouldSkipSSLValidation),
			"DD_REMOTE_CONFIGURATION_NO_TLS_VALIDATION":  pulumi.Bool(shouldSkipSSLValidation),
			"DD_LOGS_CONFIG_FORCE_USE_HTTP":              pulumi.Bool(true), // Force the use of HTTP/HTTPS rather than switching to TCP
			"DD_LOGS_CONFIG_LOGS_DD_URL":                 pulumi.Sprintf("%s", url),
			"DD_LOGS_CONFIG_LOGS_NO_SSL":                 pulumi.Bool(shouldSkipSSLValidation),
			"DD_SERVICE_DISCOVERY_FORWARDER_LOGS_DD_URL": pulumi.Sprintf("%s", url),
		}
		for key, value := range envVars {
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

func WithAdditionalFakeintake(fakeintake *fakeintake.Fakeintake) func(*Params) error {
	additionalEndpointsContentInput := fakeintake.URL.ToStringOutput().ApplyT(func(url string) (string, error) {
		endpoints := map[string][]string{
			fmt.Sprintf("%s", url): {"00000000000000000000000000000000"},
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

	// fakeintake without LB does not have a valid SSL certificate and accepts http only
	shouldEnforceHTTPInputAndSkipSSL := fakeintake.Scheme == "http"

	return func(p *Params) error {
		logsEnvVars := pulumi.Map{
			"DD_ADDITIONAL_ENDPOINTS":                   additionalEndpointsContentInput,
			"DD_LOGS_CONFIG_ADDITIONAL_ENDPOINTS":       additionalLogsEndpointsContentInput,
			"DD_SKIP_SSL_VALIDATION":                    pulumi.Bool(shouldEnforceHTTPInputAndSkipSSL),
			"DD_REMOTE_CONFIGURATION_NO_TLS_VALIDATION": pulumi.Bool(shouldEnforceHTTPInputAndSkipSSL),
			"DD_LOGS_CONFIG_LOGS_NO_SSL":                pulumi.Bool(shouldEnforceHTTPInputAndSkipSSL),
			"DD_LOGS_CONFIG_FORCE_USE_HTTP":             pulumi.Bool(true), // Force the use of HTTP/HTTPS rather than switching to TCP
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
func WithExtraComposeManifest(name string, content pulumi.StringInput) func(*Params) error {
	return func(p *Params) error {
		p.ExtraComposeManifests = append(p.ExtraComposeManifests, docker.ComposeInlineManifest{
			Name:    name,
			Content: content,
		})
		return nil
	}
}

// WithExtraComposeInlineManifest adds extra docker.ComposeInlineManifest
func WithExtraComposeInlineManifest(cpms ...docker.ComposeInlineManifest) func(*Params) error {
	return func(p *Params) error {
		p.ExtraComposeManifests = append(p.ExtraComposeManifests, cpms...)
		return nil
	}
}
