package dockervm

import (
	"github.com/DataDog/test-infra-definitions/components/datadog/agent"
	"github.com/DataDog/test-infra-definitions/components/datadog/dockeragentparams"
	resourcesAws "github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/scenarios/aws"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/fakeintake/fakeintakeparams"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/utils"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/vm/ec2os"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/vm/ec2params"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/vm/ec2vm"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Run(ctx *pulumi.Context) error {
	env, err := resourcesAws.NewEnvironment(ctx)
	if err != nil {
		return err
	}

	architecture, err := utils.GetArchitecture(env.GetCommonEnvironment())
	if err != nil {
		return err
	}

	vm, err := ec2vm.NewUnixEc2VMWithEnv(env, ec2params.WithArch(ec2os.AmazonLinuxDockerOS, architecture))
	if err != nil {
		return err
	}

	if env.AgentDeploy() {
		agentOptions := []dockeragentparams.Option{}
		if env.AgentUseFakeintake() {
			fakeIntakeOptions := []fakeintakeparams.Option{}

			if !vm.Infra.GetAwsEnvironment().InfraShouldDeployFakeintakeWithLB() {
				fakeIntakeOptions = append(fakeIntakeOptions, fakeintakeparams.WithoutLoadBalancer())
			}
			fakeintake, err := aws.NewEcsFakeintake(vm.Infra.GetAwsEnvironment(), fakeIntakeOptions...)
			if err != nil {
				return err
			}
			if !vm.Infra.GetAwsEnvironment().InfraShouldDeployFakeintakeWithLB() {
				agentOptions = append(agentOptions, dockeragentparams.WithFakeintake(fakeintake))
			} else {
				agentOptions = append(agentOptions, dockeragentparams.WithAdditionalFakeintake(fakeintake))
			}
		}
		agentOptions = append(agentOptions, dockeragentparams.WithAgentServiceEnvVariable("DD_LOG_LEVEL", pulumi.String("debug")))
		_, err = agent.NewDaemon(vm, agentOptions...)
	}

	return err
}
