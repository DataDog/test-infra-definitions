package docker

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/components/datadog/agent"
	"github.com/DataDog/test-infra-definitions/components/datadog/dockeragentparams"
	"github.com/DataDog/test-infra-definitions/components/vm"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/vm/ec2vm"

	"github.com/Masterminds/semver"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"gopkg.in/yaml.v3"
)

type Daemon struct {
	vm                 *ec2vm.EC2UnixVM
	dependsOn          pulumi.ResourceOption
	agentContainerName string
}

// NewDaemon installs the Datadog Agent through docker-compose on a VM with docker already istalled.
// It allows running multiple docker applications on the same VM. It does not check if docker is installed,
// use it with ec2os.AmazonLinuxDockerOS or install docker with `dockerManager.Install` before calling this function.
func NewDaemon(vm *ec2vm.EC2UnixVM, options ...dockeragentparams.Option) (*Daemon, error) {
	env := vm.GetCommonEnvironment()
	params, err := dockeragentparams.NewParams(options...)
	if err != nil {
		return nil, err
	}

	agentFullImagePath := dockerAgentFullImagePath(env, params.Repository)

	var composeContents []command.DockerComposeInlineManifest
	composeContents = append(composeContents, command.DockerComposeInlineManifest{
		Name:    "agent",
		Content: dockerAgentComposeContent(agentFullImagePath, env.AgentAPIKey()),
	})

	var dependOnResource pulumi.Resource
	dockerManager := vm.GetLazyDocker()

	if len(composeContents) > 0 {
		dependOnResource, err = dockerManager.ComposeStrUp("docker-on-vm", composeContents, params.EnvironmentVariables)
		if err != nil {
			return nil, err
		}
	}

	return &Daemon{
		vm:                 vm,
		dependsOn:          utils.PulumiDependsOn(dependOnResource),
		agentContainerName: agent.DefaultAgentContainerName,
	}, nil
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

func (d *Daemon) VM() vm.VM {
	return d.vm.VM
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

const (
	DefaultAgentImageRepo        = "gcr.io/datadoghq/agent"
	DefaultClusterAgentImageRepo = "gcr.io/datadoghq/cluster-agent"
	DefaultAgentContainerName    = "datadog-agent"
	defaultAgentImageTag         = "latest"
)

func dockerAgentFullImagePath(e *config.CommonEnvironment, repositoryPath string) string {
	// return agent image path if defined
	if e.AgentFullImagePath() != "" {
		return e.AgentFullImagePath()
	}

	if repositoryPath == "" {
		repositoryPath = DefaultAgentImageRepo
	}

	return utils.BuildDockerImagePath(repositoryPath, dockerAgentImageTag(e, config.AgentSemverVersion))
}

func dockerAgentImageTag(e *config.CommonEnvironment, semverVersion func(*config.CommonEnvironment) (*semver.Version, error)) string {
	// default tag
	agentImageTag := defaultAgentImageTag

	// try parse agent version
	agentVersion, err := semverVersion(e)
	if agentVersion != nil && err == nil {
		agentImageTag = agentVersion.String()
	} else {
		e.Ctx.Log.Debug("Unable to parse agent version, using latest", nil)
	}

	return agentImageTag
}
