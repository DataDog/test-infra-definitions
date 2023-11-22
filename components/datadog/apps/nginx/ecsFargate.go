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
	appName := "nginx-fg"
	namer := e.Namer.WithPrefix(appName)
	opts = append(opts, e.WithProviders(config.ProviderAWS, config.ProviderAWSX))

	EcsFargateComponent := &EcsFargateComponent{}
	if err := e.Ctx.RegisterComponentResource("dd:apps", namer.ResourceName("grp"), EcsFargateComponent, opts...); err != nil {
		return nil, err
	}

	opts = append(opts, pulumi.Parent(EcsFargateComponent))

	alb, err := lb.NewApplicationLoadBalancer(e.Ctx, namer.ResourceName("lb"), &lb.ApplicationLoadBalancerArgs{
		Name:           e.CommonNamer.DisplayName(32, pulumi.String(appName)),
		SubnetIds:      e.RandomSubnets(),
		Internal:       pulumi.BoolPtr(true),
		SecurityGroups: pulumi.ToStringArray(e.DefaultSecurityGroups()),
		DefaultTargetGroup: &lb.TargetGroupArgs{
			Name:       e.CommonNamer.DisplayName(32, pulumi.String(appName)),
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
		Name:  e.CommonNamer.DisplayName(255, pulumi.String("server")),
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
			"com.datadoghq.ad.tags": pulumi.String("[\"ecs_task_type:fargate\"]"),
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

	serverTaskDef, err := ecsClient.FargateTaskDefinitionWithAgent(e, e.CommonNamer.ResourceName("nginx-server"), e.CommonNamer.DisplayName(255, pulumi.String("server")), 1024, 2048, map[string]ecs.TaskDefinitionContainerDefinitionArgs{"server": *serverContainer}, apiKeySSMParamName, fakeintake, opts...)
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
			Subnets:        alb.LoadBalancer.Subnets(),
		},
		TaskDefinition:            serverTaskDef.TaskDefinition.Arn(),
		EnableExecuteCommand:      pulumi.BoolPtr(true),
		ContinueBeforeSteadyState: pulumi.BoolPtr(true),
		LoadBalancers: classicECS.ServiceLoadBalancerArray{
			&classicECS.ServiceLoadBalancerArgs{
				ContainerName:  pulumi.String("server"),
				ContainerPort:  pulumi.Int(80),
				TargetGroupArn: alb.DefaultTargetGroup.Arn(),
			},
		},
	}, opts...); err != nil {
		return nil, err
	}

	queryContainer := &ecs.TaskDefinitionContainerDefinitionArgs{
		Name:  e.CommonNamer.DisplayName(255, pulumi.String("query")),
		Image: pulumi.String("ghcr.io/datadog/apps-http-client:main"),
		Command: pulumi.StringArray{
			pulumi.String("-url"),
			pulumi.Sprintf("http://%s", alb.LoadBalancer.DnsName()),
		},
		Cpu:       pulumi.IntPtr(50),
		Memory:    pulumi.IntPtr(32),
		Essential: pulumi.BoolPtr(true),
		DockerLabels: pulumi.StringMap{
			"com.datadoghq.ad.tags": pulumi.String("[\"ecs_task_type:fargate\"]"),
		},
	}

	queryTaskDef, err := ecsClient.FargateTaskDefinitionWithAgent(e, e.CommonNamer.ResourceName("nginx-query"), e.CommonNamer.DisplayName(255, pulumi.String("query")), 1024, 2048, map[string]ecs.TaskDefinitionContainerDefinitionArgs{"query": *queryContainer}, apiKeySSMParamName, fakeintake, opts...)
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
