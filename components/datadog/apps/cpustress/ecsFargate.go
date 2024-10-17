package cpustress

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	fakeintakeComp "github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	ecsClient "github.com/DataDog/test-infra-definitions/resources/aws/ecs"
	classicECS "github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ecs"
	"github.com/pulumi/pulumi-awsx/sdk/v2/go/awsx/ecs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type EcsFargateComponent struct {
	pulumi.ResourceState
}

func FargateAppDefinition(e aws.Environment, clusterArn pulumi.StringInput, apiKeySSMParamName pulumi.StringInput, fakeIntake *fakeintakeComp.Fakeintake, opts ...pulumi.ResourceOption) (*EcsFargateComponent, error) {
	namer := e.Namer.WithPrefix("cpustress")
	opts = append(opts, e.WithProviders(config.ProviderAWS, config.ProviderAWSX))

	EcsFargateComponent := &EcsFargateComponent{}
	if err := e.Ctx().RegisterComponentResource("dd:apps", namer.ResourceName("grp"), EcsFargateComponent, opts...); err != nil {
		return nil, err
	}

	opts = append(opts, pulumi.Parent(EcsFargateComponent))

	stressContainer := &ecs.TaskDefinitionContainerDefinitionArgs{
		Name:  pulumi.String("stress-ng"),
		Image: pulumi.String(getStressNGImage()),
		Command: pulumi.StringArray{
			pulumi.String("--cpu=1"),
			pulumi.String("--cpu-load=15"),
		},
		Cpu:    pulumi.IntPtr(200),
		Memory: pulumi.IntPtr(64),
	}

	stressTaskDef, err := ecsClient.FargateTaskDefinitionWithAgent(e, "stress-ng", pulumi.String("stress-ng"), 1024, 2048,
		map[string]ecs.TaskDefinitionContainerDefinitionArgs{
			"stress-ng": *stressContainer,
		},
		apiKeySSMParamName,
		fakeIntake,
		opts...,
	)
	if err != nil {
		return nil, err
	}

	if _, err := ecs.NewFargateService(e.Ctx(), namer.ResourceName("stress-ng"), &ecs.FargateServiceArgs{
		Name:         e.CommonNamer().DisplayName(255, pulumi.String("stress-ng"), pulumi.String("fg")),
		Cluster:      clusterArn,
		DesiredCount: pulumi.IntPtr(1),
		NetworkConfiguration: classicECS.ServiceNetworkConfigurationArgs{
			AssignPublicIp: pulumi.BoolPtr(e.ECSServicePublicIP()),
			SecurityGroups: pulumi.ToStringArray(e.DefaultSecurityGroups()),
			Subnets:        e.RandomSubnets(),
		},
		TaskDefinition:            stressTaskDef.TaskDefinition.Arn(),
		EnableExecuteCommand:      pulumi.BoolPtr(true),
		ContinueBeforeSteadyState: pulumi.BoolPtr(true),
	}, opts...); err != nil {
		return nil, err
	}

	return EcsFargateComponent, nil
}
