package docker

import (
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/components/datadog/agent"
	"github.com/DataDog/test-infra-definitions/components/datadog/agent/dockerparams"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/vm/ec2vm"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Composer struct {
	vm                 *ec2vm.EC2UnixVM
	dependsOn          pulumi.ResourceOption
	agentContainerName string
}

// NewComposer installs docker-compose on a VM with docker already istalled. It starts compose services with
// an Agent container if called with `WithAgent` option. It allows running multiple docker applications on the same
// VM. It does not check if docker is installed, use it with ec2os.AmazonLinuxDockerOS or install docker with
// `dockerManager.Install` before calling this function.
func NewComposer(vm *ec2vm.EC2UnixVM, options ...dockerparams.Option) (*Composer, error) {
	params, err := dockerparams.NewParams(vm.GetCommonEnvironment(), options...)
	if err != nil {
		return nil, err
	}

	pulumiEnv := make(pulumi.StringMap)
	for key, value := range params.ComposeEnvVars {
		pulumiEnv[key] = pulumi.String(value)
	}

	var composeContents []command.DockerComposeInlineManifest

	if params.ComposeContent != "" {
		composeContents = append(composeContents, command.DockerComposeInlineManifest{
			Name:    "compose",
			Content: pulumi.String(params.ComposeContent),
		})
	}
	agentContainerName := ""
	if params.OptionalDockerAgentParams != nil {
		agentContainerName = "datadog-agent"

		dockerAgentParams := params.OptionalDockerAgentParams
		imagePath := dockerAgentParams.FullImagePath
		composeContents = append(composeContents, command.DockerComposeInlineManifest{
			Name:    "agent",
			Content: pulumi.Sprintf(agent.AgentComposeDefinition, imagePath, agentContainerName, vm.GetCommonEnvironment().AgentAPIKey()),
		})
		for key, value := range dockerAgentParams.Env {
			pulumiEnv[key] = pulumi.String(value)
		}
	}

	var dependOnResource pulumi.Resource
	dockerManager := vm.GetLazyDocker()

	installDockerCommand, err := dockerManager.EnsureCompose(params.PulumiResources...)
	if err != nil {
		return nil, err
	}
	if len(composeContents) > 0 {
		runCommandDeps := []pulumi.ResourceOption{utils.PulumiDependsOn(installDockerCommand)}
		dependOnResource, err = dockerManager.ComposeStrUp("docker-on-vm", composeContents, pulumiEnv, runCommandDeps...)
		if err != nil {
			return nil, err
		}
	}

	return &Composer{
		vm:                 vm,
		dependsOn:          utils.PulumiDependsOn(dependOnResource),
		agentContainerName: agentContainerName,
	}, nil
}

func (c *Composer) GetDependsOn() pulumi.ResourceOption {
	return c.dependsOn
}

func (c *Composer) GetAgentContainerName() string {
	return c.agentContainerName
}
