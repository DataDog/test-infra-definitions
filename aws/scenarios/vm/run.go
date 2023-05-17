package vm

import (
	ec2vm "github.com/DataDog/test-infra-definitions/aws/scenarios/vm/ec2VM"
	"github.com/DataDog/test-infra-definitions/datadog/agent"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Run(ctx *pulumi.Context) error {
	vm, err := ec2vm.NewEc2VM(ctx)
	if err != nil {
		return err
	}

	if vm.GetCommonEnvironment().AgentDeploy() {
		_, err = agent.NewInstaller(vm)
		return err
	}

	return nil
}
