package ec2

import (
	"github.com/DataDog/test-infra-definitions/components/datadog/agent"
	"github.com/DataDog/test-infra-definitions/components/datadog/agentparams"
	"github.com/DataDog/test-infra-definitions/components/docker"
	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/fakeintake"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func VMRun(ctx *pulumi.Context) error {
	env, err := aws.NewEnvironment(ctx)
	if err != nil {
		return err
	}

	osDesc := os.NewDescriptorFromString(env.InfraOSDescriptor())
	vm, err := NewVM(env, "vm", WithAMI(env.InfraOSImageID(), osDesc, osDesc.Architecture))
	if err != nil {
		return err
	}
	if err := vm.Export(ctx, nil); err != nil {
		return err
	}

	if env.AgentDeploy() {
		agentOptions := []agentparams.Option{}
		if env.AgentUseFakeintake() {
			fakeIntakeOptions := []fakeintake.Option{}

			if !env.InfraShouldDeployFakeintakeWithLB() {
				fakeIntakeOptions = append(fakeIntakeOptions, fakeintake.WithoutLoadBalancer())
			}

			fakeintake, err := fakeintake.NewECSFargateInstance(env, vm.Name(), fakeIntakeOptions...)
			if err != nil {
				return err
			}
			agentOptions = append(agentOptions, agentparams.WithFakeintake(fakeintake))
		}

		_, err = agent.NewHostAgent(env.CommonEnvironment, vm, agentOptions...)
		return err
	}

	return nil
}

func VMRunWithDocker(ctx *pulumi.Context) error {
	env, err := aws.NewEnvironment(ctx)
	if err != nil {
		return err
	}

	osDesc := os.NewDescriptorFromString(env.InfraOSDescriptor())
	vm, err := NewVM(env, "vm", WithAMI(env.InfraOSImageID(), osDesc, osDesc.Architecture))
	if err != nil {
		return err
	}
	if err := vm.Export(ctx, nil); err != nil {
		return err
	}

	manager := docker.NewManager(*env.CommonEnvironment, vm)
	_, err = manager.Install()
	if err != nil {
		return err
	}

	if env.AgentDeploy() {
		params := make([]agent.DockerOption, 0)
		if env.AgentFullImagePath() != "" {
			params = append(params, agent.WithAgentFullImagePath(env.AgentFullImagePath()))
		} else if env.AgentVersion() != "" {
			params = append(params, agent.WithAgentImageTag(env.AgentVersion()))
		}

		_, err = agent.NewDockerAgent(*env.CommonEnvironment, vm, manager, params...)
		if err != nil {
			return err
		}
	}

	return nil
}
