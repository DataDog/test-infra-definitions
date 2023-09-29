package fakeintake

import (
	"fmt"
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	ecsClient "github.com/DataDog/test-infra-definitions/resources/aws/ecs"
	"github.com/cenkalti/backoff/v4"
	classicECS "github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ecs"
	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/awsx"
	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/ecs"
	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/lb"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	oneSecond     = 1000
	sleepInterval = 1 * oneSecond
	maxRetries    = 120
	containerName = "fakeintake"
	port          = 80
)

type Instance struct {
	pulumi.ResourceState

	Host pulumi.StringOutput
}

func NewECSFargateInstance(e aws.Environment) (*Instance, error) {
	namer := e.Namer.WithPrefix("fakeintake")
	opts := []pulumi.ResourceOption{e.WithProviders(config.ProviderAWS, config.ProviderAWSX)}

	instance := &Instance{}
	if err := e.Ctx.RegisterComponentResource("dd:fakeintake", namer.ResourceName("grp"), instance, opts...); err != nil {
		return nil, err
	}

	opts = append(opts, pulumi.Parent(instance))

	var balancerArray classicECS.ServiceLoadBalancerArray
	var alb *lb.ApplicationLoadBalancer
	var err error
	if e.DefaultFargateLoadBalancer() {
		alb, err = lb.NewApplicationLoadBalancer(e.Ctx, namer.ResourceName("lb"), &lb.ApplicationLoadBalancerArgs{
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

		instance.Host = alb.LoadBalancer.DnsName()
		balancerArray = classicECS.ServiceLoadBalancerArray{
			&classicECS.ServiceLoadBalancerArgs{
				ContainerName:  pulumi.String(containerName),
				ContainerPort:  pulumi.Int(port),
				TargetGroupArn: alb.DefaultTargetGroup.Arn(),
			},
		}
	} else {
		instance.Host, err = FargateServiceFakeintake(e)
		if err != nil {
			return nil, err
		}
		balancerArray = classicECS.ServiceLoadBalancerArray{}
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
	if e.DefaultFargateLoadBalancer() {
		if err := e.Ctx.RegisterResourceOutputs(instance, pulumi.Map{
			"host": alb.LoadBalancer.DnsName(),
		}); err != nil {
			return nil, err
		}
	}

	return instance, nil
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
	}
}

// FargateServiceFakeintake deploys one fakeintake container to a dedicated Fargate cluster
// Hardcoded on sandbox
func FargateServiceFakeintakeWithoutLoadBalancer(e aws.Environment) (ipAddress pulumi.StringOutput, err error) {
	taskDef, err := FargateLinuxTaskDefinition(e, e.Namer.ResourceName("fakeintake-taskdef"))
	if err != nil {
		return pulumi.StringOutput{}, err
	}
	fargateService, err := ecsClient.FargateService(e, e.Namer.ResourceName("fakeintake-srv"), pulumi.String(e.ECSFargateFakeintakeClusterArn()), taskDef.TaskDefinition.Arn())
	// Hack passing taskDef.TaskDefinition.Arn() to execute apply function
	// when taskDef has an ARN, thus it is defined on AWS side
	ipAddress = pulumi.All(taskDef.TaskDefinition.Arn(), fargateService.Service.Name()).ApplyT(func(args []any) (string, error) {
		var ipAddress string
		err := backoff.Retry(func() error {
			fmt.Println("waiting for fakeintake task private ip")
			serviceName := args[1].(string)
			ecsClient, err := ecsClient.NewECSClient(e.Ctx.Context(), e.Region())
			if err != nil {
				return err
			}
			ipAddress, err = ecsClient.GetTaskPrivateIP(e.ECSFargateFakeintakeClusterArn(), serviceName)
			if err != nil {
				return err
			}
			fmt.Printf("fakeintake task private ip found: %s\n", ipAddress)
			return err
		}, backoff.WithMaxRetries(backoff.NewConstantBackOff(sleepInterval), maxRetries))
		return ipAddress, err
	}).(pulumi.StringOutput)

	return ipAddress, err
}
