package docker

import (
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/components/datadog/agent"
	"github.com/DataDog/test-infra-definitions/components/datadog/agent/dockerparams"
	"github.com/DataDog/test-infra-definitions/components/vm"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type AgentDockerInstaller struct {
	dependsOn pulumi.ResourceOption
}

func NewAgentDockerInstaller(vm *vm.UnixVM, options ...dockerparams.Option) (*AgentDockerInstaller, error) {
	commonEnv := vm.GetCommonEnvironment()
	params, err := dockerparams.NewParams(commonEnv, options...)
	if err != nil {
		return nil, err
	}

	env := make(pulumi.StringMap)
	for key, value := range params.ComposeEnvVars {
		env[key] = pulumi.String(value)
	}

	var composeContents []command.DockerComposeInlineManifest

	if params.ComposeContent != "" {
		composeContents = append(composeContents, command.DockerComposeInlineManifest{
			Name:    "compose",
			Content: pulumi.String(params.ComposeContent),
		})
	}
	if params.OptionalDockerAgentParams != nil {
		dockerAgentParams := params.OptionalDockerAgentParams
		imagePath := dockerAgentParams.FullImagePath
		composeContents = append(composeContents, command.DockerComposeInlineManifest{
			Name:    "agent",
			Content: pulumi.Sprintf(agent.AgentComposeDefinition, imagePath, commonEnv.AgentAPIKey()),
		})
		for key, value := range dockerAgentParams.Env {
			env[key] = pulumi.String(value)
		}
	}

	var dependOnResource pulumi.Resource
	dockerManager := vm.GetLazyDocker()
	if len(composeContents) > 0 {
		dependOnResource, err = dockerManager.ComposeStrUp("docker-on-vm", composeContents, env, params.PulumiResources...)
	} else {
		dependOnResource, err = dockerManager.Install()
	}

	if err != nil {
		return nil, err
	}

	return &AgentDockerInstaller{dependsOn: utils.PulumiDependsOn(dependOnResource)}, nil
}

func (d *AgentDockerInstaller) GetDependsOn() pulumi.ResourceOption {
	return d.dependsOn
}
