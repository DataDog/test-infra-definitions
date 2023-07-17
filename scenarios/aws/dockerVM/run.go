package dockervm

import (
	"github.com/DataDog/test-infra-definitions/components/datadog/agent/docker"
	resourcesAws "github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/utils"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Run(ctx *pulumi.Context) error {
	env, err := resourcesAws.NewEnvironment(ctx)
	if err != nil {
		return err
	}

	var options []func(*docker.Params) error
	if env.AgentDeploy() {
		options = append(options, docker.WithAgent())
	}

	architecture, err := utils.GetArchitecture(env.GetCommonEnvironment())
	if err != nil {
		return err
	}

	options = append(options, docker.WithArchitecture(architecture))

	_, err = docker.NewAgentDockerInstallerWithEnv(env, options...)

	return err
}
