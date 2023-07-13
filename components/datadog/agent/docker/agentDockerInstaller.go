package docker

import (
	"fmt"
	"strings"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/components/datadog/agent"
	"github.com/DataDog/test-infra-definitions/components/os"
	resourcesAws "github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/vm/ec2os"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/vm/ec2params"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/vm/ec2vm"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type AgentDockerInstaller struct {
	dependsOn pulumi.ResourceOption
}

func NewAgentDockerInstaller(env resourcesAws.Environment, options ...func(*Params) error) (*AgentDockerInstaller, error) {
	commonEnv := env.GetCommonEnvironment()
	architecture, err := getArchitecture(commonEnv)
	if err != nil {
		return nil, err
	}

	vm, err := ec2vm.NewUnixEc2VMWithEnv(env, ec2params.WithArch(ec2os.AmazonLinuxDockerOS, architecture))
	if err != nil {
		return nil, err
	}

	params, err := newParams(commonEnv, options...)
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

func (d *AgentDockerInstaller) GetDependsOn() pulumi.ResourceOption {
	return d.dependsOn
}

func getArchitecture(commonEnv *config.CommonEnvironment) (os.Architecture, error) {
	var arch os.Architecture
	archStr := strings.ToLower(commonEnv.InfraOSArchitecture())
	switch archStr {
	case "x86_64":
		arch = os.AMD64Arch
	case "arm64":
		arch = os.ARM64Arch
	case "":
		arch = os.AMD64Arch // Default
	default:
		return arch, fmt.Errorf("the architecture type '%v' is not valid", archStr)
	}
	return arch, nil
}
