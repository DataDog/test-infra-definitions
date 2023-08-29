package fakeintake

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	localECS "github.com/DataDog/test-infra-definitions/resources/aws/ecs"
	classicECS "github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ecs"
	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/awsx"
	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/ecs"
	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/lb"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"os"
)

const (
	containerName = "fakeintake"
	port          = 80
)

type Instance struct {
	pulumi.ResourceState

	Host pulumi.StringOutput
}

func FargateLinuxTaskDefinition(e aws.Environment, name string) (*ecs.FargateTaskDefinition, error) {
	return ecs.NewFargateTaskDefinition(e.Ctx, e.Namer.ResourceName(name), &ecs.FargateTaskDefinitionArgs{
		Containers: map[string]ecs.TaskDefinitionContainerDefinitionArgs{
			containerName: *fargateLinuxContainerDefinition(),
		},
		Cpu:    pulumi.StringPtr("256"),
		Memory: pulumi.StringPtr("512"),
		ExecutionRole: &awsx.DefaultRoleWithPolicyArgs{
			RoleArn: pulumi.StringPtr(e.ECSTaskExecutionRole()),
		},
		TaskRole: &awsx.DefaultRoleWithPolicyArgs{
			RoleArn: pulumi.StringPtr(e.ECSTaskRole()),
		},
		Family: e.CommonNamer.DisplayName(13, pulumi.String("fakeintake-ecs")),
	}, e.WithProviders(config.ProviderAWS, config.ProviderAWSX))
}

func fargateLinuxContainerDefinition() *ecs.TaskDefinitionContainerDefinitionArgs {
	return &ecs.TaskDefinitionContainerDefinitionArgs{
		Name:        pulumi.StringPtr(containerName),
		Image:       pulumi.StringPtr("public.ecr.aws/datadog/fakeintake:latest"),
		Essential:   pulumi.BoolPtr(true),
		MountPoints: ecs.TaskDefinitionMountPointArray{},
		Environment: ecs.TaskDefinitionKeyValuePairArray{},
		PortMappings: ecs.TaskDefinitionPortMappingArray{
			ecs.TaskDefinitionPortMappingArgs{
				ContainerPort: pulumi.Int(port),
				HostPort:      pulumi.Int(port),
				Protocol:      pulumi.StringPtr("tcp"),
			},
		},
		VolumesFrom: ecs.TaskDefinitionVolumeFromArray{},
	}
}

func NewECSFargateInstance(e aws.Environment) (*Instance, error) {
	namer := e.Namer.WithPrefix("fakeintake")
	opts := []pulumi.ResourceOption{e.WithProviders(config.ProviderAWS, config.ProviderAWSX)}

	instance := &Instance{}
	if err := e.Ctx.RegisterComponentResource("dd:fakeintake", namer.ResourceName("grp"), instance, opts...); err != nil {
		return nil, err
	}

	opts = append(opts, pulumi.Parent(instance))

	alb, err := lb.NewApplicationLoadBalancer(e.Ctx, namer.ResourceName("lb"), &lb.ApplicationLoadBalancerArgs{
		Name:           e.CommonNamer.DisplayName(32, pulumi.String("fakeintake")),
		SubnetIds:      e.RandomSubnets(),
		Internal:       pulumi.BoolPtr(!e.ECSServicePublicIP()),
		SecurityGroups: pulumi.ToStringArray(e.DefaultSecurityGroups()),
		DefaultTargetGroup: &lb.TargetGroupArgs{
			Name:       e.CommonNamer.DisplayName(32, pulumi.String("fakeintake")),
			Port:       pulumi.IntPtr(port),
			Protocol:   pulumi.StringPtr("HTTP"),
			TargetType: pulumi.StringPtr("ip"),
			VpcId:      pulumi.StringPtr(e.DefaultVPCID()),
		},
		Listener: &lb.ListenerArgs{
			Port:     pulumi.IntPtr(port),
			Protocol: pulumi.StringPtr("HTTP"),
		},
	}, opts...)
	if err != nil {
		return nil, err
	}
	var balancerArray classicECS.ServiceLoadBalancerArray
	if os.Getenv("DD_DISABLE_LB") != "" {
		var FargateErr error
		instance.Host, FargateErr = localECS.FargateServiceFakeintake(e)
		if FargateErr != nil {
			return nil, FargateErr
		}
		balancerArray = classicECS.ServiceLoadBalancerArray{}
	} else {
		instance.Host = alb.LoadBalancer.DnsName()
		balancerArray = classicECS.ServiceLoadBalancerArray{
			&classicECS.ServiceLoadBalancerArgs{
				ContainerName:  pulumi.String(containerName),
				ContainerPort:  pulumi.Int(port),
				TargetGroupArn: alb.DefaultTargetGroup.Arn(),
			},
		}
	}
	if _, err := ecs.NewFargateService(e.Ctx, namer.ResourceName("srv"), &ecs.FargateServiceArgs{
		Cluster:              pulumi.StringPtr(e.ECSFargateFakeintakeClusterArn()),
		Name:                 e.CommonNamer.DisplayName(255, pulumi.String("fakeintake")),
		DesiredCount:         pulumi.IntPtr(1),
		EnableExecuteCommand: pulumi.BoolPtr(true),
		NetworkConfiguration: &classicECS.ServiceNetworkConfigurationArgs{
			AssignPublicIp: pulumi.BoolPtr(false),
			SecurityGroups: pulumi.ToStringArray(e.DefaultSecurityGroups()),
			Subnets:        e.RandomSubnets(),
		},
		LoadBalancers: balancerArray,
		TaskDefinitionArgs: &ecs.FargateServiceTaskDefinitionArgs{
			Containers: map[string]ecs.TaskDefinitionContainerDefinitionArgs{
				containerName: {
					Name:        pulumi.String(containerName),
					Image:       pulumi.String("public.ecr.aws/datadog/fakeintake:latest"),
					Essential:   pulumi.BoolPtr(true),
					MountPoints: ecs.TaskDefinitionMountPointArray{},
					Environment: ecs.TaskDefinitionKeyValuePairArray{},
					PortMappings: ecs.TaskDefinitionPortMappingArray{
						ecs.TaskDefinitionPortMappingArgs{
							ContainerPort: pulumi.Int(port),
							HostPort:      pulumi.Int(port),
							Protocol:      pulumi.StringPtr("tcp"),
						},
					},
					VolumesFrom: ecs.TaskDefinitionVolumeFromArray{},
				},
			},
			Cpu:    pulumi.StringPtr("256"),
			Memory: pulumi.StringPtr("512"),
			ExecutionRole: &awsx.DefaultRoleWithPolicyArgs{
				RoleArn: pulumi.StringPtr(e.ECSTaskExecutionRole()),
			},
			TaskRole: &awsx.DefaultRoleWithPolicyArgs{
				RoleArn: pulumi.StringPtr(e.ECSTaskRole()),
			},
			Family: e.CommonNamer.DisplayName(255, pulumi.String("fakeintake-ecs")),
		},
		ContinueBeforeSteadyState: pulumi.BoolPtr(true),
	}, opts...); err != nil {
		return nil, err
	}

	if err := e.Ctx.RegisterResourceOutputs(instance, pulumi.Map{
		"host": alb.LoadBalancer.DnsName(),
	}); err != nil {
		return nil, err
	}

	return instance, nil
}
