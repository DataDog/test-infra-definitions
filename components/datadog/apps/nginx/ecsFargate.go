package nginx

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	ddfakeintake "github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
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

func FargateAppDefinition(e aws.Environment, clusterArn pulumi.StringInput, apiKeySSMParamName pulumi.StringInput, fakeintake *ddfakeintake.ConnectionExporter, opts ...pulumi.ResourceOption) (*EcsFargateComponent, error) {
	namer := e.Namer.WithPrefix("nginx").WithPrefix("fg")
	opts = append(opts, e.WithProviders(config.ProviderAWS, config.ProviderAWSX))

	EcsFargateComponent := &EcsFargateComponent{}
	if err := e.Ctx.RegisterComponentResource("dd:apps", namer.ResourceName("grp"), EcsFargateComponent, opts...); err != nil {
		return nil, err
	}

	opts = append(opts, pulumi.Parent(EcsFargateComponent))

	alb, err := lb.NewApplicationLoadBalancer(e.Ctx, namer.ResourceName("lb"), &lb.ApplicationLoadBalancerArgs{
		Name:           e.CommonNamer.DisplayName(32, pulumi.String("nginx"), pulumi.String("fg")),
		SubnetIds:      e.RandomSubnets(),
		Internal:       pulumi.BoolPtr(true),
		SecurityGroups: pulumi.ToStringArray(e.DefaultSecurityGroups()),
		DefaultTargetGroup: &lb.TargetGroupArgs{
			Name:       e.CommonNamer.DisplayName(32, pulumi.String("nginx"), pulumi.String("fg")),
			Port:       pulumi.IntPtr(80),
			Protocol:   pulumi.StringPtr("HTTP"),
			TargetType: pulumi.StringPtr("ip"),
		},
		Listener: &lb.ListenerArgs{
			Port:     pulumi.IntPtr(80),
			Protocol: pulumi.StringPtr("HTTP"),
		},
	}, opts...)
	if err != nil {
		return nil, err
	}

	serverContainer := &ecs.TaskDefinitionContainerDefinitionArgs{
		Name:  pulumi.String("nginx"),
		Image: pulumi.String("ghcr.io/datadog/apps-nginx-server:main"),
		DockerLabels: pulumi.StringMap{
			"com.datadoghq.ad.checks": pulumi.String(utils.JSONMustMarshal(
				map[string]interface{}{
					"nginx": map[string]interface{}{
						"init_config": map[string]interface{}{},
						"instances": []map[string]interface{}{
							{
								"nginx_status_url": "http://%%host%%/nginx_status",
							},
						},
					},
				},
			)),
			"com.datadoghq.ad.tags": pulumi.String("[\"ecs_launch_type:fargate\"]"),
		},
		Cpu:       pulumi.IntPtr(100),
		Memory:    pulumi.IntPtr(96),
		Essential: pulumi.BoolPtr(true),
		DependsOn: ecs.TaskDefinitionContainerDependencyArray{
			ecs.TaskDefinitionContainerDependencyArgs{
				ContainerName: pulumi.String("datadog-agent"),
				Condition:     pulumi.String("HEALTHY"),
			},
		},
		PortMappings: ecs.TaskDefinitionPortMappingArray{
			ecs.TaskDefinitionPortMappingArgs{
				ContainerPort: pulumi.IntPtr(80),
				HostPort:      pulumi.IntPtr(80),
				Protocol:      pulumi.StringPtr("tcp"),
			},
		},
		HealthCheck: ecs.TaskDefinitionHealthCheckArgs{
			Command: pulumi.StringArray{
				pulumi.String("CMD-SHELL"),
				pulumi.String("apk add curl && curl --fail http://localhost || exit 1"),
			},
		},
		LogConfiguration: ecsClient.GetFirelensLogConfiguration(pulumi.String("nginx"), pulumi.String("nginx"), apiKeySSMParamName),
	}

	serverTaskDef, err := ecsClient.FargateTaskDefinitionWithAgent(e, "nginx", pulumi.String("nginx-fg"), 1024, 2048, map[string]ecs.TaskDefinitionContainerDefinitionArgs{"nginx": *serverContainer}, apiKeySSMParamName, fakeintake, opts...)
	if err != nil {
		return nil, err
	}

	if _, err := ecs.NewFargateService(e.Ctx, namer.ResourceName("server"), &ecs.FargateServiceArgs{
		Cluster:      clusterArn,
		Name:         e.CommonNamer.DisplayName(255, pulumi.String("nginx"), pulumi.String("fg")),
		DesiredCount: pulumi.IntPtr(1),
		NetworkConfiguration: classicECS.ServiceNetworkConfigurationArgs{
			AssignPublicIp: pulumi.BoolPtr(e.ECSServicePublicIP()),
			SecurityGroups: pulumi.ToStringArray(e.DefaultSecurityGroups()),
			Subnets:        alb.LoadBalancer.Subnets(),
		},
		TaskDefinition:            serverTaskDef.TaskDefinition.Arn(),
		EnableExecuteCommand:      pulumi.BoolPtr(true),
		ContinueBeforeSteadyState: pulumi.BoolPtr(true),
		LoadBalancers: classicECS.ServiceLoadBalancerArray{
			&classicECS.ServiceLoadBalancerArgs{
				ContainerName:  pulumi.String("nginx"),
				ContainerPort:  pulumi.Int(80),
				TargetGroupArn: alb.DefaultTargetGroup.Arn(),
			},
		},
	}, opts...); err != nil {
		return nil, err
	}

	queryContainer := &ecs.TaskDefinitionContainerDefinitionArgs{
		Name:  pulumi.String("query"),
		Image: pulumi.String("ghcr.io/datadog/apps-http-client:main"),
		Command: pulumi.StringArray{
			pulumi.String("-url"),
			pulumi.Sprintf("http://%s", alb.LoadBalancer.DnsName()),
		},
		Cpu:       pulumi.IntPtr(50),
		Memory:    pulumi.IntPtr(32),
		Essential: pulumi.BoolPtr(true),
	}

	queryTaskDef, err := ecsClient.FargateTaskDefinitionWithAgent(e, "nginx-query", pulumi.String("nginx-fg-query"), 1024, 2048, map[string]ecs.TaskDefinitionContainerDefinitionArgs{"query": *queryContainer}, apiKeySSMParamName, fakeintake, opts...)
	if err != nil {
		return nil, err
	}

	if _, err := ecs.NewFargateService(e.Ctx, namer.ResourceName("query"), &ecs.FargateServiceArgs{
		Cluster:      clusterArn,
		Name:         e.CommonNamer.DisplayName(255, pulumi.ToStringArray([]string{"nginx", "fg", "query"})...),
		DesiredCount: pulumi.IntPtr(1),
		NetworkConfiguration: classicECS.ServiceNetworkConfigurationArgs{
			AssignPublicIp: pulumi.BoolPtr(e.ECSServicePublicIP()),
			SecurityGroups: pulumi.ToStringArray(e.DefaultSecurityGroups()),
			Subnets:        alb.LoadBalancer.Subnets(),
		},
		TaskDefinition:            queryTaskDef.TaskDefinition.Arn(),
		EnableExecuteCommand:      pulumi.BoolPtr(true),
		ContinueBeforeSteadyState: pulumi.BoolPtr(true),
	}, opts...); err != nil {
		return nil, err
	}

	return EcsFargateComponent, nil
}
