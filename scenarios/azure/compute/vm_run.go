package compute

import (
	"github.com/DataDog/test-infra-definitions/components/datadog/agent"
	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/resources/azure"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func VMRun(ctx *pulumi.Context) error {
	env, err := azure.NewEnvironment(ctx)
	if err != nil {
		return err
	}

	osDesc := os.DescriptorFromString(env.InfraOSDescriptor(), os.UbuntuDefault)
	vm, err := NewVM(env, "vm", WithImageURN(env.InfraOSImageID(), osDesc, osDesc.Architecture))
	if err != nil {
		return err
	}
	if err := vm.Export(ctx, nil); err != nil {
		return err
	}

	if env.AgentDeploy() {
		_, err = agent.NewHostAgent(&env, vm)
		return err
	}

	return nil
}
