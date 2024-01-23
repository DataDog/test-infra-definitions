package nginx

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
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
	namer := e.Namer.WithPrefix("nginx").WithPrefix("ec2")
	opts = append(opts, e.WithProviders(config.ProviderAWS, config.ProviderAWSX))

	ecsComponent := &EcsComponent{}
	if err := e.Ctx.RegisterComponentResource("dd:apps", namer.ResourceName("grp"), ecsComponent, opts...); err != nil {
		return nil, err
	}

	opts = append(opts, pulumi.Parent(ecsComponent))

	alb, err := lb.NewApplicationLoadBalancer(e.Ctx, namer.ResourceName("lb"), &lb.ApplicationLoadBalancerArgs{
		Name:           e.CommonNamer.DisplayName(32, pulumi.String("nginx"), pulumi.String("ec2")),
		SubnetIds:      e.RandomSubnets(),
		Internal:       pulumi.BoolPtr(true),
		SecurityGroups: pulumi.ToStringArray(e.DefaultSecurityGroups()),
		DefaultTargetGroup: &lb.TargetGroupArgs{
			Name:       e.CommonNamer.DisplayName(32, pulumi.String("nginx"), pulumi.String("ec2")),
			Port:       pulumi.IntPtr(80),
			Protocol:   pulumi.StringPtr("HTTP"),
			TargetType: pulumi.StringPtr("instance"),
			VpcId:      pulumi.StringPtr(e.DefaultVPCID()),
		},
		Listener: &lb.ListenerArgs{
			Port:     pulumi.IntPtr(80),
			Protocol: pulumi.StringPtr("HTTP"),
		},
	}, opts...)
	if err != nil {
		return nil, err
	}

	if _, err := ecs.NewEC2Service(e.Ctx, namer.ResourceName("server"), &ecs.EC2ServiceArgs{
		Name:                 e.CommonNamer.DisplayName(255, pulumi.String("nginx"), pulumi.String("ec2")),
		Cluster:              clusterArn,
		DesiredCount:         pulumi.IntPtr(2),
		EnableExecuteCommand: pulumi.BoolPtr(true),
		TaskDefinitionArgs: &ecs.EC2ServiceTaskDefinitionArgs{
			Containers: map[string]ecs.TaskDefinitionContainerDefinitionArgs{
				"nginx": {
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
						"com.datadoghq.ad.tags": pulumi.String("[\"ecs_launch_type:ec2\"]"),
					},
					Cpu:    pulumi.IntPtr(100),
					Memory: pulumi.IntPtr(96),
					MountPoints: ecs.TaskDefinitionMountPointArray{
						ecs.TaskDefinitionMountPointArgs{
							SourceVolume:  pulumi.StringPtr("cache"),
							ContainerPath: pulumi.StringPtr("/var/cache/nginx"),
						},
						ecs.TaskDefinitionMountPointArgs{
							SourceVolume:  pulumi.StringPtr("var-run"),
							ContainerPath: pulumi.StringPtr("/var/run"),
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
						Interval: pulumi.Int(30),
						Retries:  pulumi.Int(3),
						Timeout:  pulumi.Int(5),
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
			Family:      e.CommonNamer.DisplayName(255, pulumi.ToStringArray([]string{"nginx", "ec2"})...),
			Volumes: classicECS.TaskDefinitionVolumeArray{
				classicECS.TaskDefinitionVolumeArgs{
					Name: pulumi.String("cache"),
					DockerVolumeConfiguration: classicECS.TaskDefinitionVolumeDockerVolumeConfigurationArgs{
						Scope: pulumi.StringPtr("task"),
					},
				},
				classicECS.TaskDefinitionVolumeArgs{
					Name: pulumi.String("var-run"),
					DockerVolumeConfiguration: classicECS.TaskDefinitionVolumeDockerVolumeConfigurationArgs{
						Scope: pulumi.StringPtr("task"),
					},
				},
			},
		},
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

	if _, err := ecs.NewEC2Service(e.Ctx, namer.ResourceName("query"), &ecs.EC2ServiceArgs{
		Name:                 e.CommonNamer.DisplayName(255, pulumi.ToStringArray([]string{"nginx", "ec2", "query"})...),
		Cluster:              clusterArn,
		DesiredCount:         pulumi.IntPtr(1),
		EnableExecuteCommand: pulumi.BoolPtr(true),
		TaskDefinitionArgs: &ecs.EC2ServiceTaskDefinitionArgs{
			Containers: map[string]ecs.TaskDefinitionContainerDefinitionArgs{
				"query": {
					Name:  pulumi.String("query"),
					Image: pulumi.String("ghcr.io/datadog/apps-http-client:main"),
					Command: pulumi.StringArray{
						pulumi.String("-url"),
						pulumi.Sprintf("http://%s", alb.LoadBalancer.DnsName()),
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
			Family:      e.CommonNamer.DisplayName(255, pulumi.ToStringArray([]string{"nginx", "ec2", "query"})...),
		},
	}, opts...); err != nil {
		return nil, err
	}

	return ecsComponent, nil
}
