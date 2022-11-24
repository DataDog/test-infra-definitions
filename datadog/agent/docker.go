package agent

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/command"
	"github.com/DataDog/test-infra-definitions/common/config"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	agentComposeDefinition = `version: "3.9"
services:
  agent:
    image: %s
    volumes:
      - "/var/run/docker.sock:/var/run/docker.sock"
      - "/proc/:/host/proc"
      - "/sys/fs/cgroup/:/host/sys/fs/cgroup"
    environment:
      DD_API_KEY: %s
      DD_PROCESS_AGENT_ENABLED: true
      DD_DOGSTATSD_NON_LOCAL_TRAFFIC: true`
	defaultAgentImageRepo = "gcr.io/datadoghq/agent"
	defaultAgentImageTag  = "latest"
)

func DockerFullImagePath(e config.CommonEnvironment) string {
	// return agent image path if defined
	if e.AgentFullImagePath() != "" {
		return e.AgentFullImagePath()
	}

	return fmt.Sprintf("%s:%s", defaultAgentImageRepo, DockerImageTag(e))
}

func DockerImageTag(e config.CommonEnvironment) string {
	// default tag
	agentImageTag := defaultAgentImageTag

	// try parse agent version
	agentVersion, err := config.AgentSemverVersion(e)
	if agentVersion != nil && err == nil {
		agentImageTag = agentVersion.String()
	} else {
		e.Ctx.Log.Debug("Unable to parse Agent version, using latest", nil)
	}

	return agentImageTag
}

func NewDockerInstallation(e config.CommonEnvironment, dockerManager *command.DockerManager, extraConfiguration pulumi.StringInput) (*remote.Command, error) {
	composeContents := []command.DockerComposeInlineManifest{
		{
			Name:    "agent",
			Content: pulumi.Sprintf(agentComposeDefinition, DockerFullImagePath(e), e.AgentAPIKey()),
		},
	}
	if extraConfiguration != nil {
		composeContents = append(composeContents, command.DockerComposeInlineManifest{Name: "agent-custom", Content: extraConfiguration})
	}

	return dockerManager.ComposeStrUp("agent", composeContents)
}

func NewDockerInstallationWithAgentImage(e config.CommonEnvironment, dockerManager *command.DockerManager, agentImagePath, extraConfiguration string) (*remote.Command, error) {
	fmt.Printf("NewDockerInstallationWithAgentImage with agent %s and extraCompose: %s\n", agentImagePath, extraConfiguration)
	composeContents := []command.DockerComposeInlineManifest{
		{
			Name:    "agent",
			Content: pulumi.Sprintf(agentComposeDefinition, agentImagePath, e.AgentAPIKey()),
		},
	}
	if len(extraConfiguration) > 0 {
		fmt.Printf("With extraCompose: %v\n", extraConfiguration)
		composeContents = append(composeContents, command.DockerComposeInlineManifest{Name: "agent-custom", Content: pulumi.String(extraConfiguration)})
	}

	return dockerManager.ComposeStrUp("agent", composeContents)
}
