package docker

import (
	"github.com/DataDog/test-infra-definitions/command"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/common/vm"
	"github.com/DataDog/test-infra-definitions/datadog/agent"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type OnVM struct {
	vm        *vm.UbuntuVM
	dependsOn pulumi.ResourceOption
}

func NewOnVM(ctx *pulumi.Context, vm *vm.UbuntuVM, options ...func(*Params) error) (*OnVM, error) {
	commonEnv := vm.GetCommonEnvironment()
	params, err := newParams(commonEnv, options...)
	if err != nil {
		return nil, err
	}

	runner := vm.GetRunner()
	packageManager := vm.GetAptManager()
	dockerManager := command.NewDockerManager(runner, packageManager)
	env := make(pulumi.StringMap)
	for key, value := range params.composeEnvVars {
		env[key] = pulumi.String(value)
	}

	var composeContents []command.DockerComposeInlineManifest

	if params.composeContent != "" {
		composeContents = append(composeContents, command.DockerComposeInlineManifest{
			Name:    "compose",
			Content: pulumi.String(params.composeContent),
		})
	}
	if params.optionalDockerAgentParams != nil {
		dockerAgentParams := params.optionalDockerAgentParams
		imagePath := dockerAgentParams.fullImagePath
		composeContents = append(composeContents, command.DockerComposeInlineManifest{
			Name:    "agent",
			Content: pulumi.Sprintf(agent.AgentComposeDefinition, imagePath, commonEnv.AgentAPIKey()),
		})
		for key, value := range dockerAgentParams.env {
			env[key] = pulumi.String(value)
		}
	}

	var dependOnResource pulumi.Resource
	if len(composeContents) > 0 {
		dependOnResource, err = dockerManager.ComposeStrUp("docker-on-vm", composeContents, env, params.pulumiResources...)
	} else {
		dependOnResource, err = dockerManager.Install()
	}

	if err != nil {
		return nil, err
	}

	return &OnVM{vm: vm, dependsOn: utils.PulumiDependsOn(dependOnResource)}, nil
}

func (d *OnVM) GetDependsOn() pulumi.ResourceOption {
	return d.dependsOn
}

func (d *OnVM) GetVM() *vm.UbuntuVM {
	return d.vm
}
