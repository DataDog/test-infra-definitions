package computerun

import (
	"github.com/DataDog/test-infra-definitions/components/datadog/agent"
	"github.com/DataDog/test-infra-definitions/components/datadog/agentparams"
	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/resources/azure"
	"github.com/DataDog/test-infra-definitions/scenarios/azure/compute"
	"github.com/DataDog/test-infra-definitions/scenarios/azure/fakeintake"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func VMRun(ctx *pulumi.Context) error {
	env, err := azure.NewEnvironment(ctx)
	if err != nil {
		return err
	}

	osDesc := os.DescriptorFromString(env.InfraOSDescriptor(), os.UbuntuDefault)
	vm, err := compute.NewVM(env, "vm", compute.WithImageURN(env.InfraOSImageID(), osDesc, osDesc.Architecture))
	if err != nil {
		return err
	}
	if err := vm.Export(ctx, nil); err != nil {
		return err
	}

	if env.AgentDeploy() {
		agentOptions := []agentparams.Option{}
		if env.AgentUseFakeintake() {
			fakeintake, err := fakeintake.NewVMInstance(env)
			if err != nil {
				return err
			}
			if err := fakeintake.Export(ctx, nil); err != nil {
				return err
			}

			agentOptions = append(agentOptions, agentparams.WithFakeintake(fakeintake))
		}
		_, err = agent.NewHostAgent(&env, vm, agentOptions...)
		return err
	}

	return nil
}