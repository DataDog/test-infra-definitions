package aspnetsample

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	fakeintakeComp "github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	ecsClient "github.com/DataDog/test-infra-definitions/resources/aws/ecs"
	classicECS "github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ecs"
	"github.com/pulumi/pulumi-awsx/sdk/v2/go/awsx/ecs"
	"github.com/pulumi/pulumi-awsx/sdk/v2/go/awsx/lb"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type EcsFargateComponent struct {
	pulumi.ResourceState
}

func FargateAppDefinition(e aws.Environment, clusterArn pulumi.StringInput, apiKeySSMParamName pulumi.StringInput, fakeIntake *fakeintakeComp.Fakeintake, opts ...pulumi.ResourceOption) (*EcsFargateComponent, error) {
	namer := e.Namer.WithPrefix("aspnetsample").WithPrefix("fg")

	opts = append(opts, e.WithProviders(config.ProviderAWS, config.ProviderAWSX))

	EcsFargateComponent := &EcsFargateComponent{}
	if err := e.Ctx.RegisterComponentResource("dd:apps", namer.ResourceName("grp"), EcsFargateComponent, opts...); err != nil {
		return nil, err
	}

	opts = append(opts, pulumi.Parent(EcsFargateComponent))

	nlb, err := lb.NewNetworkLoadBalancer(e.Ctx, namer.ResourceName("lb"), &lb.NetworkLoadBalancerArgs{
		Name:      e.CommonNamer.DisplayName(32, pulumi.String("aspnetsample"), pulumi.String("fg")),
		SubnetIds: e.RandomSubnets(),
		Internal:  pulumi.BoolPtr(true),
		DefaultTargetGroup: &lb.TargetGroupArgs{
			Name:       e.CommonNamer.DisplayName(32, pulumi.String("aspnetsample"), pulumi.String("fg")),
			Port:       pulumi.IntPtr(80),
			Protocol:   pulumi.StringPtr("TCP"),
			TargetType: pulumi.StringPtr("ip"),
			VpcId:      pulumi.StringPtr(e.DefaultVPCID()),
		},
		Listener: &lb.ListenerArgs{
			Port:     pulumi.IntPtr(80),
			Protocol: pulumi.StringPtr("TCP"),
		},
	}, opts...)
	if err != nil {
		return nil, err
	}

	serverContainer := &ecs.TaskDefinitionContainerDefinitionArgs{
		Name:  pulumi.String("aspnetsample"),
		Image: pulumi.String("mcr.microsoft.com/dotnet/samples:aspnetapp-nanoserver-ltsc2022"),
		DockerLabels: pulumi.StringMap{
			"com.datadoghq.ad.checks": pulumi.String(utils.JSONMustMarshal(
				map[string]interface{}{
					"http_check": map[string]interface{}{
						"name":        "aspnetsample",
						"init_config": map[string]interface{}{},
						"instances": []map[string]interface{}{
							{
								"url": "http://%%host%%/80",
							},
						},
					},
				},
			)),
			"com.datadoghq.ad.tags": pulumi.String("[\"ecs_launch_type:fargate\"]"),
		},
		Cpu:       pulumi.IntPtr(1024),
		Memory:    pulumi.IntPtr(2048),
		Essential: pulumi.BoolPtr(true),
		// Health check is disabled in the agent.
		//DependsOn: ecs.TaskDefinitionContainerDependencyArray{
		//	ecs.TaskDefinitionContainerDependencyArgs{
		//		ContainerName: pulumi.String("datadog-agent"),
		//		Condition:     pulumi.String("HEALTHY"),
		//	},
		//},
		PortMappings: ecs.TaskDefinitionPortMappingArray{
			ecs.TaskDefinitionPortMappingArgs{
				ContainerPort: pulumi.IntPtr(80),
				HostPort:      pulumi.IntPtr(80),
				Protocol:      pulumi.StringPtr("tcp"),
			},
		},
	}

	serverTaskDef, err := ecsClient.FargateWindowsTaskDefinitionWithAgent(e, "aspnet-fg-server", pulumi.String("aspnet-fg"), 1024, 2048, map[string]ecs.TaskDefinitionContainerDefinitionArgs{"aspnetsample": *serverContainer}, apiKeySSMParamName, fakeIntake, opts...)
	if err != nil {
		return nil, err
	}

	if _, err := ecs.NewFargateService(e.Ctx, namer.ResourceName("server"), &ecs.FargateServiceArgs{
		Cluster:      clusterArn,
		Name:         e.CommonNamer.DisplayName(255, pulumi.String("aspnetsample"), pulumi.String("fg")),
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
				ContainerName:  pulumi.String("aspnetsample"),
				ContainerPort:  pulumi.Int(80),
				TargetGroupArn: nlb.DefaultTargetGroup.Arn(),
			},
		},
	}, opts...); err != nil {
		return nil, err
	}

	return EcsFargateComponent, nil
}
