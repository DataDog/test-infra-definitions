package agent

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components"
	"github.com/DataDog/test-infra-definitions/components/docker"
	remoteComp "github.com/DataDog/test-infra-definitions/components/remote"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	agentComposeDefinition = `version: "3.9"
services:
  agent:
    image: %s
    container_name: %s
    volumes:
      - "/var/run/docker.sock:/var/run/docker.sock"
      - "/proc/:/host/proc"
      - "/sys/fs/cgroup/:/host/sys/fs/cgroup"
    environment:
      DD_API_KEY: %s
      DD_PROCESS_AGENT_ENABLED: true
      DD_DOGSTATSD_NON_LOCAL_TRAFFIC: true`
)

// DockerAgent is a Docker installer on a remote Host
type DockerAgent struct {
	pulumi.ResourceState
	components.Component
}

func NewDockerAgent(e config.CommonEnvironment, vm *remoteComp.Host, manager *docker.Manager, options ...DockerOption) (*DockerAgent, error) {
	return components.NewComponent(e, vm.Name(), func(comp *DockerAgent) error {
		params, err := newDockerParams(options...)
		if err != nil {
			return err
		}
		defaultAgentParams(params)

		// Create environment variables passed to Agent container
		envVars := make(pulumi.StringMap)
		for key, value := range params.composeEnvVars {
			envVars[key] = pulumi.String(value)
		}

		// We can have multiple compose files in compose.
		composeContents := []docker.ComposeInlineManifest{
			{
				Name:    "agent",
				Content: pulumi.Sprintf(agentComposeDefinition, params.fullImagePath, "datadog-agent", e.AgentAPIKey()),
			},
		}
		if params.composeContent != "" {
			composeContents = append(composeContents, docker.ComposeInlineManifest{
				Name:    "agent-custom",
				Content: pulumi.String(params.composeContent),
			})
		}

		_, err = manager.ComposeStrUp("agent", composeContents, envVars, pulumi.Parent(comp))
		return err
	})
}

func defaultAgentParams(params *dockerParams) {
	if params.fullImagePath == "" {
		params.fullImagePath = utils.BuildDockerImagePath(defaultAgentImageRepo, defaultAgentImageTag)
	}
}
