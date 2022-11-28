package dockerVM

import (
	"github.com/DataDog/test-infra-definitions/aws/scenarios/vm/vm"
	"github.com/DataDog/test-infra-definitions/datadog/agent"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Run(ctx *pulumi.Context) error {
	vm, err := vm.NewVM(ctx)
	if err != nil {
		return err
	}

	// TODO add basic command to DockerManager
	_, err = agent.NewDockerAgentInstallation(vm.Environment, vm.DockerManager, "", "")
	return err
}
