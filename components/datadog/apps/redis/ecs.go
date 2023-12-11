package redis

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/resources/aws"

	classicECS "github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ecs"
	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/awsx"
	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/ecs"
	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/lb"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type EcsComponent struct {
	pulumi.ResourceState
}

func EcsAppDefinition(e aws.Environment, clusterArn pulumi.StringInput, opts ...pulumi.ResourceOption) (*EcsComponent, error) {
	namer := e.Namer.WithPrefix("redis")
	opts = append(opts, e.WithProviders(config.ProviderAWS, config.ProviderAWSX))

	ecsComponent := &EcsComponent{}
	if err := e.Ctx.RegisterComponentResource("dd:apps", namer.ResourceName("grp"), ecsComponent, opts...); err != nil {
		return nil, err
	}

	opts = append(opts, pulumi.Parent(ecsComponent))

	nlb, err := lb.NewNetworkLoadBalancer(e.Ctx, namer.ResourceName("lb"), &lb.NetworkLoadBalancerArgs{
		Name:      e.CommonNamer.DisplayName(32, pulumi.String("redis")),
		SubnetIds: e.RandomSubnets(),
		Internal:  pulumi.BoolPtr(true),
		DefaultTargetGroup: &lb.TargetGroupArgs{
			Name:       e.CommonNamer.DisplayName(32, pulumi.String("redis")),
			Port:       pulumi.IntPtr(6379),
			Protocol:   pulumi.StringPtr("TCP"),
			TargetType: pulumi.StringPtr("instance"),
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

	if _, err := ecs.NewEC2Service(e.Ctx, namer.ResourceName("server"), &ecs.EC2ServiceArgs{
		Name:                 e.CommonNamer.DisplayName(255, pulumi.String("redis")),
		Cluster:              clusterArn,
		DesiredCount:         pulumi.IntPtr(2),
		EnableExecuteCommand: pulumi.BoolPtr(true),
		TaskDefinitionArgs: &ecs.EC2ServiceTaskDefinitionArgs{
			Containers: map[string]ecs.TaskDefinitionContainerDefinitionArgs{
				"redis": {
					Name:  pulumi.String("redis"),
					Image: pulumi.String("redis:latest"),
					Command: pulumi.StringArray{
						pulumi.String("--loglevel"),
						pulumi.String("verbose"),
					},
					Cpu:    pulumi.IntPtr(100),
					Memory: pulumi.IntPtr(32),
					PortMappings: ecs.TaskDefinitionPortMappingArray{
						ecs.TaskDefinitionPortMappingArgs{
							ContainerPort: pulumi.IntPtr(6379),
							HostPort:      pulumi.IntPtr(6379),
							Protocol:      pulumi.StringPtr("tcp"),
						},
					},
				},
			},
			ExecutionRole: &awsx.DefaultRoleWithPolicyArgs{
				RoleArn: pulumi.StringPtr(e.ECSTaskExecutionRole()),
			},
			TaskRole: &awsx.DefaultRoleWithPolicyArgs{
				RoleArn: pulumi.StringPtr(e.ECSTaskRole()),
			},
			NetworkMode: pulumi.StringPtr("bridge"),
			Family:      e.CommonNamer.DisplayName(255, pulumi.String("redis-ec2")),
		},
		LoadBalancers: classicECS.ServiceLoadBalancerArray{
			&classicECS.ServiceLoadBalancerArgs{
				ContainerName:  pulumi.String("redis"),
				ContainerPort:  pulumi.Int(6379),
				TargetGroupArn: nlb.DefaultTargetGroup.Arn(),
			},
		},
	}, opts...); err != nil {
		return nil, err
	}

	if _, err := ecs.NewEC2Service(e.Ctx, namer.ResourceName("query"), &ecs.EC2ServiceArgs{
		Name:                 e.CommonNamer.DisplayName(255, pulumi.String("redis-query")),
		Cluster:              clusterArn,
		DesiredCount:         pulumi.IntPtr(1),
		EnableExecuteCommand: pulumi.BoolPtr(true),
		TaskDefinitionArgs: &ecs.EC2ServiceTaskDefinitionArgs{
			Containers: map[string]ecs.TaskDefinitionContainerDefinitionArgs{
				"query": {
					Name:  pulumi.String("query"),
					Image: pulumi.String("ghcr.io/datadog/apps-redis-client:main"),
					Command: pulumi.StringArray{
						pulumi.String("-addr"),
						pulumi.Sprintf("%s:6379", nlb.LoadBalancer.DnsName()),
					},
					Cpu:    pulumi.IntPtr(50),
					Memory: pulumi.IntPtr(32),
				},
			},
			ExecutionRole: &awsx.DefaultRoleWithPolicyArgs{
				RoleArn: pulumi.StringPtr(e.ECSTaskExecutionRole()),
			},
			TaskRole: &awsx.DefaultRoleWithPolicyArgs{
				RoleArn: pulumi.StringPtr(e.ECSTaskRole()),
			},
			NetworkMode: pulumi.StringPtr("bridge"),
			Family:      e.CommonNamer.DisplayName(255, pulumi.String("redis-query-ec2")),
		},
	}, opts...); err != nil {
		return nil, err
	}

	return ecsComponent, nil
}
