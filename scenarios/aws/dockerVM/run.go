package dockervm

import (
	"github.com/DataDog/test-infra-definitions/components/datadog/agent"
	resourcesAws "github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/utils"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/vm/ec2os"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/vm/ec2params"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/vm/ec2vm"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Run(ctx *pulumi.Context) error {
	env, err := resourcesAws.NewEnvironment(ctx)
	if err != nil {
		return err
	}

	architecture, err := utils.GetArchitecture(env.GetCommonEnvironment())
	if err != nil {
		return err
	}

	vm, err := ec2vm.NewUnixEc2VMWithEnv(env, ec2params.WithArch(ec2os.AmazonLinuxDockerOS, architecture))
	if err != nil {
		return err
	}

	if env.AgentDeploy() {
		_, err = agent.NewDaemon(vm)
	}

	return err
}
