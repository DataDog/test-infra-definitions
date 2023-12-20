package ec2

import (
	"github.com/DataDog/test-infra-definitions/components/datadog/agent"
	"github.com/DataDog/test-infra-definitions/components/datadog/agentparams"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/dogstatsd"
	"github.com/DataDog/test-infra-definitions/components/datadog/dockeragentparams"
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

			if env.InfraShouldDeployFakeintakeWithLB() {
				fakeIntakeOptions = append(fakeIntakeOptions, fakeintake.WithLoadBalancer())
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

	manager, _, err := docker.NewManager(*env.CommonEnvironment, vm, true)
	if err != nil {
		return err
	}

	if env.AgentDeploy() {
		agentOptions := make([]dockeragentparams.Option, 0)
		if env.AgentFullImagePath() != "" {
			agentOptions = append(agentOptions, dockeragentparams.WithFullImagePath(env.AgentFullImagePath()))
		} else if env.AgentVersion() != "" {
			agentOptions = append(agentOptions, dockeragentparams.WithImageTag(env.AgentVersion()))
		}

		if env.AgentUseFakeintake() {
			fakeIntakeOptions := []fakeintake.Option{}

			if env.InfraShouldDeployFakeintakeWithLB() {
				fakeIntakeOptions = append(fakeIntakeOptions, fakeintake.WithLoadBalancer())
			}

			fakeintake, err := fakeintake.NewECSFargateInstance(env, vm.Name(), fakeIntakeOptions...)
			if err != nil {
				return err
			}
			agentOptions = append(agentOptions, dockeragentparams.WithFakeintake(fakeintake))
		}

		_, err = agent.NewDockerAgent(*env.CommonEnvironment, vm, manager, agentOptions...)
		if err != nil {
			return err
		}
	}

	if env.TestingWorkloadDeploy() {
		_, err := manager.ComposeStrUp("dogstatsd-apps", []docker.ComposeInlineManifest{dogstatsd.DockerComposeManifest}, pulumi.StringMap{"HOST_IP": vm.Address})
		if err != nil {
			return err
		}
	}

	return nil
}
