package vm

import (
	"github.com/DataDog/test-infra-definitions/aws/scenarios/ecs"
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
