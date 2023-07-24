package dockervm

import (
	"github.com/DataDog/test-infra-definitions/components/datadog/agent/docker"
	"github.com/DataDog/test-infra-definitions/components/datadog/agent/dockerparams"
	resourcesAws "github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/utils"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Run(ctx *pulumi.Context) error {
	env, err := resourcesAws.NewEnvironment(ctx)
	if err != nil {
		return err
	}

	var options []dockerparams.Option
	if env.AgentDeploy() {
		options = append(options, dockerparams.WithAgent())
	}

	architecture, err := utils.GetArchitecture(env.GetCommonEnvironment())
	if err != nil {
		return err
	}

	options = append(options, dockerparams.WithArchitecture(architecture))

	_, err = docker.NewDaemonWithEnv(env, options...)

	return err
}
