package docker

import (
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/components/datadog/agent"
	"github.com/DataDog/test-infra-definitions/components/datadog/agent/dockerparams"
	"github.com/DataDog/test-infra-definitions/components/os"
	resourcesAws "github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/vm/ec2os"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/vm/ec2params"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/vm/ec2vm"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"gopkg.in/yaml.v3"
)

type Daemon struct {
	vm                 *ec2vm.EC2UnixVM
	dependsOn          pulumi.ResourceOption
	agentContainerName string
}

func NewDaemonWithEnv(env resourcesAws.Environment, options ...dockerparams.Option) (*Daemon, error) {
	commonEnv := env.GetCommonEnvironment()
	params, err := dockerparams.NewParams(commonEnv, options...)
	if err != nil {
		return nil, err
	}

	vm, err := ec2vm.NewUnixEc2VMWithEnv(env, ec2params.WithArch(ec2os.AmazonLinuxDockerOS, params.Architecture))
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
		dockerAgentParams := params.OptionalDockerAgentParams
		imagePath := dockerAgentParams.FullImagePath
		composeContents = append(composeContents, command.DockerComposeInlineManifest{
			Name:    "agent",
			Content: dockerAgentComposeContent(imagePath, commonEnv.AgentAPIKey()),
		})
		for key, value := range dockerAgentParams.Env {
			pulumiEnv[key] = pulumi.String(value)
		}
	}

	var dependOnResource pulumi.Resource
	dockerManager := vm.GetLazyDocker()
	if len(composeContents) > 0 {
		runCommandDeps := params.PulumiResources
		dependOnResource, err = dockerManager.ComposeStrUp("docker-on-vm", composeContents, pulumiEnv, runCommandDeps...)
	} else {
		dependOnResource, err = dockerManager.EnsureCompose()
	}

	if err != nil {
		return nil, err
	}

	return &Daemon{
		vm:                 vm,
		dependsOn:          utils.PulumiDependsOn(dependOnResource),
		agentContainerName: agentContainerName}, nil
}

func NewDaemon(ctx *pulumi.Context, options ...func(*dockerparams.Params) error) (*Daemon, error) {
	env, err := resourcesAws.NewEnvironment(ctx)
	if err != nil {
		return nil, err
	}
	return NewDaemonWithEnv(env, options...)
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

func (d *Daemon) GetOS() os.OS {
	return d.vm.GetOS()
}

func dockerAgentComposeContent(agentImagePath string, apiKey pulumi.StringInput) pulumi.StringOutput {
	agentManifestContent := pulumi.All(apiKey).ApplyT(func(args []interface{}) (string, error) {
		apiKeyResolved := args[0].(string)
		agentManifest := command.DockerComposeManifest{
			Version: "3.9",
			Services: map[string]command.DockerComposeManifestService{
				"agent": {
					Image:         agentImagePath,
					ContainerName: agent.DefaultAgentContainerName,
					Volumes: []string{
						"/var/run/docker.sock:/var/run/docker.sock",
						"/proc/:/host/proc",
						"/sys/fs/cgroup/:/host/sys/fs/cgroup",
					},
					Environment: map[string]any{
						"DD_API_KEY":                     apiKeyResolved,
						"DD_PROCESS_AGENT_ENABLED":       true,
						"DD_DOGSTATSD_NON_LOCAL_TRAFFIC": true,
					},
				},
			},
		}
		data, err := yaml.Marshal(agentManifest)
		return string(data), err
	}).(pulumi.StringOutput)

	return agentManifestContent
}
