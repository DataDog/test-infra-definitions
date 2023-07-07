package docker

import (
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/components/datadog/agent"
	"github.com/DataDog/test-infra-definitions/components/datadog/agent/dockerparams"
	"github.com/DataDog/test-infra-definitions/components/vm"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Daemon struct {
	vm                 *vm.UnixVM
	dependsOn          pulumi.ResourceOption
	agentContainerName string
}

func NewDaemon(vm *vm.UnixVM, options ...dockerparams.Option) (*Daemon, error) {
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
	agentContainerName := ""
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
		agentContainerName = "docker-on-vm-compose-tmp-agent-1" // TODO: Improve the naming and make it more robust
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

	return &Daemon{
		vm:                 vm,
		dependsOn:          utils.PulumiDependsOn(dependOnResource),
		agentContainerName: agentContainerName}, nil
}

func (d *Daemon) GetDependsOn() pulumi.ResourceOption {
	return d.dependsOn
}

func (d *Daemon) GetAgentContainerName() string {
	return d.agentContainerName
}

type ClientData struct {
	Connection utils.Connection
}

func (d *Daemon) Deserialize(result auto.UpResult) (*ClientData, error) {
	vmData, err := d.vm.Deserialize(result)
	if err != nil {
		return nil, err
	}

	return &ClientData{Connection: vmData.Connection}, nil
}
