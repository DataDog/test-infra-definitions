package redis

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	fakeintakeComp "github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	ecsClient "github.com/DataDog/test-infra-definitions/resources/aws/ecs"
	classicECS "github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ecs"
	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/ecs"
	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/lb"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type EcsFargateComponent struct {
	pulumi.ResourceState
}

func FargateAppDefinition(e aws.Environment, clusterArn pulumi.StringInput, apiKeySSMParamName pulumi.StringInput, fakeIntake *fakeintakeComp.Fakeintake, opts ...pulumi.ResourceOption) (*EcsFargateComponent, error) {
	appName := "redis-fg"
	namer := e.Namer.WithPrefix(appName)
	opts = append(opts, e.WithProviders(config.ProviderAWS, config.ProviderAWSX))

	EcsFargateComponent := &EcsFargateComponent{}
	if err := e.Ctx.RegisterComponentResource("dd:apps", namer.ResourceName("grp"), EcsFargateComponent, opts...); err != nil {
		return nil, err
	}

	opts = append(opts, pulumi.Parent(EcsFargateComponent))

	nlb, err := lb.NewNetworkLoadBalancer(e.Ctx, namer.ResourceName("lb"), &lb.NetworkLoadBalancerArgs{
		Name:      e.CommonNamer.DisplayName(32, pulumi.String(appName)),
		SubnetIds: e.RandomSubnets(),
		Internal:  pulumi.BoolPtr(true),
		DefaultTargetGroup: &lb.TargetGroupArgs{
			Name:       e.CommonNamer.DisplayName(32, pulumi.String(appName)),
			Port:       pulumi.IntPtr(6379),
			Protocol:   pulumi.StringPtr("TCP"),
			TargetType: pulumi.StringPtr("ip"),
			VpcId:      pulumi.StringPtr(e.DefaultVPCID()),
		},
		Listener: &lb.ListenerArgs{
			Port:     pulumi.IntPtr(6379),
			Protocol: pulumi.StringPtr("TCP"),
		},
	}, opts...)
	if err != nil {
		return nil, err
	}

	serverContainer := &ecs.TaskDefinitionContainerDefinitionArgs{
		Name:      e.CommonNamer.DisplayName(255, pulumi.String("server")),
		Image:     pulumi.String("redis:latest"),
		Cpu:       pulumi.IntPtr(0),
		Essential: pulumi.BoolPtr(true),
		DependsOn: ecs.TaskDefinitionContainerDependencyArray{
			ecs.TaskDefinitionContainerDependencyArgs{
				ContainerName: pulumi.String("datadog-agent"),
				Condition:     pulumi.String("HEALTHY"),
			},
		},
		PortMappings: ecs.TaskDefinitionPortMappingArray{
			ecs.TaskDefinitionPortMappingArgs{
				ContainerPort: pulumi.IntPtr(6379),
				HostPort:      pulumi.IntPtr(6379),
				Protocol:      pulumi.StringPtr("tcp"),
			},
		},
		DockerLabels: pulumi.StringMap{
			"com.datadoghq.ad.tags": pulumi.String("[\"ecs_task_type:fargate\"]"),
		},
		LogConfiguration: ecsClient.GetFirelensLogConfiguration(pulumi.String("redis"), pulumi.String("redis"), apiKeySSMParamName),
	}

	serverTaskDef, err := ecsClient.FargateTaskDefinitionWithAgent(e, e.CommonNamer.ResourceName("server"), e.CommonNamer.DisplayName(255, pulumi.String("server")), 1024, 2048, map[string]ecs.TaskDefinitionContainerDefinitionArgs{"server": *serverContainer}, apiKeySSMParamName, fakeIntake, opts...)
	if err != nil {
		return nil, err
	}

	if _, err := ecs.NewFargateService(e.Ctx, namer.ResourceName("server"), &ecs.FargateServiceArgs{
		Cluster:      clusterArn,
		Name:         namer.DisplayName(255, pulumi.String("server")),
		DesiredCount: pulumi.IntPtr(1),
		NetworkConfiguration: classicECS.ServiceNetworkConfigurationArgs{
			AssignPublicIp: pulumi.BoolPtr(e.ECSServicePublicIP()),
			SecurityGroups: pulumi.ToStringArray(e.DefaultSecurityGroups()),
			Subnets:        nlb.LoadBalancer.Subnets(),
		},
		TaskDefinition:            serverTaskDef.TaskDefinition.Arn(),
		EnableExecuteCommand:      pulumi.BoolPtr(true),
		ContinueBeforeSteadyState: pulumi.BoolPtr(true),
		LoadBalancers: classicECS.ServiceLoadBalancerArray{
			&classicECS.ServiceLoadBalancerArgs{
				ContainerName:  pulumi.String("server"),
				ContainerPort:  pulumi.Int(6379),
				TargetGroupArn: nlb.DefaultTargetGroup.Arn(),
			},
		},
	}, opts...); err != nil {
		return nil, err
	}

	queryContainer := &ecs.TaskDefinitionContainerDefinitionArgs{
		Name:  e.CommonNamer.DisplayName(255, pulumi.String("query")),
		Image: pulumi.String("ghcr.io/datadog/apps-redis-client:main"),
		Command: pulumi.StringArray{
			pulumi.String("-addr"),
			pulumi.Sprintf("%s:6379", nlb.LoadBalancer.DnsName()),
		},
		Cpu:       pulumi.IntPtr(50),
		Memory:    pulumi.IntPtr(32),
		Essential: pulumi.BoolPtr(true),
		DockerLabels: pulumi.StringMap{
			"com.datadoghq.ad.tags": pulumi.String("[\"ecs_task_type:fargate\"]"),
		},
	}

	queryTaskDef, err := ecsClient.FargateTaskDefinitionWithAgent(e, e.CommonNamer.ResourceName("query"), e.CommonNamer.DisplayName(255, pulumi.String("query")), 1024, 2048, map[string]ecs.TaskDefinitionContainerDefinitionArgs{"query": *queryContainer}, apiKeySSMParamName, fakeIntake, opts...)
	if err != nil {
		return nil, err
	}

	if _, err := ecs.NewFargateService(e.Ctx, namer.ResourceName("query"), &ecs.FargateServiceArgs{
		Cluster:      clusterArn,
		Name:         namer.DisplayName(255, pulumi.String("query")),
		DesiredCount: pulumi.IntPtr(1),
		NetworkConfiguration: classicECS.ServiceNetworkConfigurationArgs{
			AssignPublicIp: pulumi.BoolPtr(e.ECSServicePublicIP()),
			SecurityGroups: pulumi.ToStringArray(e.DefaultSecurityGroups()),
			Subnets:        nlb.LoadBalancer.Subnets(),
		},
		TaskDefinition:            queryTaskDef.TaskDefinition.Arn(),
		EnableExecuteCommand:      pulumi.BoolPtr(true),
		ContinueBeforeSteadyState: pulumi.BoolPtr(true),
	}, opts...); err != nil {
		return nil, err
	}

	return EcsFargateComponent, nil
}
