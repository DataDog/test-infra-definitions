package dockerVM

import (
	"github.com/DataDog/test-infra-definitions/aws/ec2/ec2"
	"github.com/DataDog/test-infra-definitions/datadog/agent"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Run(ctx *pulumi.Context) error {
	vm, err := ec2.NewVM(ctx)
	if err != nil {
		return err
	}
	_, err = agent.NewDockerAgentInstallation(vm.CommonEnvironment, vm.DockerManager, "", nil)
	return err
}
