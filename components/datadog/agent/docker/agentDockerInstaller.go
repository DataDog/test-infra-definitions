package docker

import (
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/components/datadog/agent"
	resourcesAws "github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/vm/ec2os"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/vm/ec2params"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/vm/ec2vm"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type AgentDockerInstaller struct {
	dependsOn pulumi.ResourceOption
}

func NewAgentDockerInstallerWithEnv(env resourcesAws.Environment, options ...func(*Params) error) (*AgentDockerInstaller, error) {

	commonEnv := env.GetCommonEnvironment()
	params, err := newParams(commonEnv, options...)
	if err != nil {
		return nil, err
	}

	vm, err := ec2vm.NewUnixEc2VMWithEnv(env, ec2params.WithArch(ec2os.AmazonLinuxDockerOS, params.architecture))
	if err != nil {
		return nil, err
	}

	pulumiEnv := make(pulumi.StringMap)
	for key, value := range params.composeEnvVars {
		pulumiEnv[key] = pulumi.String(value)
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
			pulumiEnv[key] = pulumi.String(value)
		}
	}

	var dependOnResource pulumi.Resource
	dockerManager := vm.GetLazyDocker()
	if len(composeContents) > 0 {
		runCommandDeps := params.pulumiResources
		dependOnResource, err = dockerManager.ComposeStrUp("docker-on-vm", composeContents, pulumiEnv, runCommandDeps...)
	} else {
		dependOnResource, err = dockerManager.InstallCompose()
	}

	if err != nil {
		return nil, err
	}

	return &AgentDockerInstaller{dependsOn: utils.PulumiDependsOn(dependOnResource)}, nil
}

func NewAgentDockerInstaller(ctx *pulumi.Context, options ...func(*Params) error) (*AgentDockerInstaller, error) {
	env, err := resourcesAws.NewEnvironment(ctx)
	if err != nil {
		return nil, err
	}
	return NewAgentDockerInstallerWithEnv(env, options...)
}

func (d *AgentDockerInstaller) GetDependsOn() pulumi.ResourceOption {
	return d.dependsOn
}
