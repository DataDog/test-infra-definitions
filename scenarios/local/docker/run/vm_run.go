package localdockerrun

import (
	"github.com/DataDog/test-infra-definitions/components/datadog/agent"
	"github.com/DataDog/test-infra-definitions/components/datadog/agentparams"
	"github.com/DataDog/test-infra-definitions/resources/local"
	localdocker "github.com/DataDog/test-infra-definitions/scenarios/local/docker"

	"github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func VMRun(ctx *pulumi.Context) error {
	env, err := local.NewEnvironment(ctx)
	if err != nil {
		return err
	}

	vm, err := localdocker.NewVM(env, "vm")
	if err != nil {
		return err
	}
	if err := vm.Export(ctx, nil); err != nil {
		return err
	}

	if env.AgentDeploy() {
		agentOptions := []agentparams.Option{}
		if env.AgentUseFakeintake() {
			fakeintake, err := fakeintake.NewLocalDockerFakeintake(&env, "fakeintake")
			if err != nil {
				return err
			}
			err = fakeintake.Export(ctx, nil)
			if err != nil {
				return err
			}
			agentOptions = append(agentOptions, agentparams.WithFakeintake(fakeintake))
		}
		agentOptions = append(agentOptions, agentparams.WithHostname("localdocker-vm"))
		_, err = agent.NewHostAgent(&env, vm, agentOptions...)
		return err
	}

	return nil
}
