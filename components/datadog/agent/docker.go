package agent

import (
	"strings"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"gopkg.in/yaml.v3"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components"
	"github.com/DataDog/test-infra-definitions/components/datadog/dockeragentparams"
	"github.com/DataDog/test-infra-definitions/components/docker"
	remoteComp "github.com/DataDog/test-infra-definitions/components/remote"
)

const (
	agentContainerName = "datadog-agent"
)

type DockerAgentOutput struct {
	components.JSONImporter

	ContainerName string `json:"containerName"`
}

// DockerAgent is a Docker installer on a remote Host
type DockerAgent struct {
	pulumi.ResourceState
	components.Component

	ContainerName pulumi.StringOutput `pulumi:"containerName"`
}

func (h *DockerAgent) Export(ctx *pulumi.Context, out *DockerAgentOutput) error {
	return components.Export(ctx, h, out)
}

func NewDockerAgent(e config.CommonEnvironment, vm *remoteComp.Host, manager *docker.Manager, options ...dockeragentparams.Option) (*DockerAgent, error) {
	return components.NewComponent(e, vm.Name(), func(comp *DockerAgent) error {
		params, err := dockeragentparams.NewParams(&e, options...)
		if err != nil {
			return err
		}
		defaultAgentParams(params)

		// We can have multiple compose files in compose.
		composeContents := []docker.ComposeInlineManifest{dockerAgentComposeManifest(params.FullImagePath, e.AgentAPIKey(), params.AgentServiceEnvironment)}
		composeContents = append(composeContents, params.ExtraComposeManifests...)

		_, err = manager.ComposeStrUp("agent", composeContents, params.EnvironmentVariables, pulumi.Parent(comp))
		if err != nil {
			return err
		}

		// Fill component
		comp.ContainerName = pulumi.String(agentContainerName).ToStringOutput()
		return nil
	})
}

func dockerAgentComposeManifest(agentImagePath string, apiKey pulumi.StringInput, envVars pulumi.Map) docker.ComposeInlineManifest {
	runInPrivileged := false
	for k := range envVars {
		if strings.HasPrefix(k, "DD_SYSTEM_PROBE_") {
			runInPrivileged = true
			break
		}
	}

	agentManifestContent := pulumi.All(apiKey, envVars).ApplyT(func(args []interface{}) (string, error) {
		apiKeyResolved := args[0].(string)
		envVarsResolved := args[1].(map[string]any)
		agentManifest := docker.ComposeManifest{
			Version: "3.9",
			Services: map[string]docker.ComposeManifestService{
				"agent": {
					Privileged:    runInPrivileged,
					Image:         agentImagePath,
					ContainerName: agentContainerName,
					Volumes: []string{
						"/var/run/docker.sock:/var/run/docker.sock",
						"/proc/:/host/proc",
						"/sys/fs/cgroup/:/host/sys/fs/cgroup",
						"/var/run/datadog:/var/run/datadog",
						"/sys/kernel/tracing:/sys/kernel/tracing",
					},
					Environment: map[string]any{
						"DD_API_KEY": apiKeyResolved,
						// DD_PROCESS_CONFIG_PROCESS_COLLECTION_ENABLED is compatible with Agent 7.35+
						"DD_PROCESS_CONFIG_PROCESS_COLLECTION_ENABLED": true,
					},
					Pid:   "host",
					Ports: []string{"8125:8125/udp", "8126:8126/tcp"},
				},
			},
		}
		for key, value := range envVarsResolved {
			agentManifest.Services["agent"].Environment[key] = value
		}
		data, err := yaml.Marshal(agentManifest)
		return string(data), err
	}).(pulumi.StringOutput)

	return docker.ComposeInlineManifest{
		Name:    "agent",
		Content: agentManifestContent,
	}
}

func defaultAgentParams(params *dockeragentparams.Params) {
	if params.FullImagePath != "" {
		return
	}

	if params.Repository == "" {
		params.Repository = defaultAgentImageRepo
	}
	if params.ImageTag == "" {
		params.ImageTag = defaultAgentImageTag
	}
	params.FullImagePath = utils.BuildDockerImagePath(params.Repository, params.ImageTag)
}
