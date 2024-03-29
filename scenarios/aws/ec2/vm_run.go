package ec2

import (
	"errors"

	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/datadog/agent"
	"github.com/DataDog/test-infra-definitions/components/datadog/agentparams"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/dogstatsd"
	"github.com/DataDog/test-infra-definitions/components/datadog/dockeragentparams"
	"github.com/DataDog/test-infra-definitions/components/datadog/updater"
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

	osDesc := os.DescriptorFromString(env.InfraOSDescriptor(), os.AmazonLinuxECSDefault)
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

	if env.UpdaterDeploy() {
		if env.AgentDeploy() {
			return errors.New("cannot deploy both agent and updater installers, updater installs the agent")
		}

		_, err := updater.NewHostUpdater(env.CommonEnvironment, vm)
		return err
	}

	return nil
}

func VMRunWithDocker(ctx *pulumi.Context) error {
	env, err := aws.NewEnvironment(ctx)
	if err != nil {
		return err
	}

	// If no OS is provided, we default to AmazonLinuxECS as it ships with Docker pre-installed
	osDesc := os.DescriptorFromString(env.InfraOSDescriptor(), os.AmazonLinuxECSDefault)
	vm, err := NewVM(env, "vm", WithAMI(env.InfraOSImageID(), osDesc, osDesc.Architecture))
	if err != nil {
		return err
	}
	if err := vm.Export(ctx, nil); err != nil {
		return err
	}

	installEcrCredsHelperCmd, err := InstallECRCredentialsHelper(env, vm)
	if err != nil {
		return err
	}

	manager, _, err := docker.NewManager(*env.CommonEnvironment, vm, utils.PulumiDependsOn(installEcrCredsHelperCmd))
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

			if err := fakeintake.Export(env.Ctx, nil); err != nil {
				return err
			}
			if err != nil {
				return err
			}

			agentOptions = append(agentOptions, dockeragentparams.WithFakeintake(fakeintake))
		}

		if env.TestingWorkloadDeploy() {
			agentOptions = append(agentOptions, dockeragentparams.WithExtraComposeManifest(dogstatsd.DockerComposeManifest.Name, dogstatsd.DockerComposeManifest.Content))
			agentOptions = append(agentOptions, dockeragentparams.WithEnvironmentVariables(pulumi.StringMap{"HOST_IP": vm.Address}))
		}

		_, err = agent.NewDockerAgent(*env.CommonEnvironment, vm, manager, agentOptions...)
		if err != nil {
			return err
		}
	}

	return nil
}
