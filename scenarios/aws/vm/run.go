package vm

import (
	"github.com/DataDog/test-infra-definitions/components/datadog/agent"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/ecs"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/vm/ec2vm"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Run(ctx *pulumi.Context) error {
	vm, err := ec2vm.NewEc2VM(ctx)
	if err != nil {
		return err
	}

	if vm.GetCommonEnvironment().AgentDeploy() {
		agentOptions := []func(*agent.Params) error{}
		if vm.GetCommonEnvironment().AgentUseFakeintake() {
			fakeintake, err := ecs.NewEcsFakeintake(vm.Infra)
			if err != nil {
				return err
			}
			agentOptions = append(agentOptions, agent.WithFakeintake(fakeintake))
		}

		_, err = agent.NewInstaller(vm, agentOptions...)
		return err
	}

	return nil
}
