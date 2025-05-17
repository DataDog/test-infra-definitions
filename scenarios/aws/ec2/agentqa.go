package ec2

import (
	"errors"

	"github.com/DataDog/test-infra-definitions/components/datadog/agent"
	"github.com/DataDog/test-infra-definitions/components/datadog/agentparams"
	"github.com/DataDog/test-infra-definitions/components/datadog/updater"
	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/fakeintake"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type agentQAContext struct {
	pulumiContext *pulumi.Context
	env           aws.Environment
}

// Will create the Agent QA Environment
func AgentQARun(ctx *pulumi.Context) error {
	env, err := aws.NewEnvironment(ctx)
	if err != nil {
		return err
	}

	qaContext := &agentQAContext{
		pulumiContext: ctx,
		env:           env,
	}

	_, err = newClientNode(qaContext)
	if err != nil {
		return err
	}

	return nil
}

// A client node is a host VM where the agent is installed
// TODO: Return pulumi resource instead of int
func newClientNode(qa *agentQAContext) (*int, error) {
	// TODO: Os version
	osDesc := os.WindowsServer2022
	vm, err := NewVM(qa.env, "vm", WithAMI(qa.env.InfraOSImageID(), osDesc, osDesc.Architecture))
	if err != nil {
		return nil, err
	}
	if err := vm.Export(qa.pulumiContext, nil); err != nil {
		return nil, err
	}

	// TODO: This has been copy-pasted from VMRun, we should refactor this
	if qa.env.AgentDeploy() {
		// TODO: Configure options differently for different nodes
		agentOptions := []agentparams.Option{}
		if qa.env.AgentUseFakeintake() {
			fakeIntakeOptions := []fakeintake.Option{}

			if qa.env.InfraShouldDeployFakeintakeWithLB() {
				fakeIntakeOptions = append(fakeIntakeOptions, fakeintake.WithLoadBalancer())
			}

			fakeintake, err := fakeintake.NewECSFargateInstance(qa.env, vm.Name(), fakeIntakeOptions...)
			if err != nil {
				return nil, err
			}
			agentOptions = append(agentOptions, agentparams.WithFakeintake(fakeintake))
		}

		if qa.env.AgentFlavor() != "" {
			agentOptions = append(agentOptions, agentparams.WithFlavor(qa.env.AgentFlavor()))
		}

		if qa.env.AgentConfigPath() != "" {
			configContent, err := qa.env.CustomAgentConfig()
			if err != nil {
				return nil, err
			}
			agentOptions = append(agentOptions, agentparams.WithAgentConfig(configContent))
		}

		agent, err := agent.NewHostAgent(&qa.env, vm, agentOptions...)
		if err != nil {
			return nil, err
		}

		err = agent.Export(qa.pulumiContext, nil)
		if err != nil {
			return nil, err
		}
	}

	if qa.env.UpdaterDeploy() {
		if qa.env.AgentDeploy() {
			return nil, errors.New("cannot deploy both agent and updater installers, updater installs the agent")
		}

		_, err := updater.NewHostUpdater(&qa.env, vm)
		return nil, err
	}

	return nil, nil
}
