package agent

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/resources/aws"

	classicECS "github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ecs"
	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/awsx"
	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/ecs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func ECSLinuxDaemonDefinition(e aws.Environment, name string, apiKeySSMParamName, clusterArn pulumi.StringInput) (*ecs.EC2Service, error) {
	return ecs.NewEC2Service(e.Ctx, e.Namer.ResourceName(name), &ecs.EC2ServiceArgs{
		Name:               e.CommonNamer.DisplayName(pulumi.String(name)),
		Cluster:            clusterArn,
		SchedulingStrategy: pulumi.StringPtr("DAEMON"),
		PlacementConstraints: classicECS.ServicePlacementConstraintArray{
			classicECS.ServicePlacementConstraintArgs{
				Type:       pulumi.String("memberOf"),
				Expression: pulumi.StringPtr("attribute:ecs.os-type == linux"),
			},
		},
		NetworkConfiguration: classicECS.ServiceNetworkConfigurationArgs{
			AssignPublicIp: pulumi.BoolPtr(false),
			SecurityGroups: pulumi.ToStringArray(e.DefaultSecurityGroups()),
			Subnets:        pulumi.ToStringArray(e.DefaultSubnets()),
		},
		EnableExecuteCommand: pulumi.BoolPtr(true),
		TaskDefinitionArgs: &ecs.EC2ServiceTaskDefinitionArgs{
			Containers: map[string]ecs.TaskDefinitionContainerDefinitionArgs{
				"datadog-agent": ecsLinuxAgentSingleContainerDefinition(*e.CommonEnvironment, apiKeySSMParamName),
			},
			ExecutionRole: &awsx.DefaultRoleWithPolicyArgs{
				RoleArn: pulumi.StringPtr(e.ECSTaskExecutionRole()),
			},
			TaskRole: &awsx.DefaultRoleWithPolicyArgs{
				RoleArn: pulumi.StringPtr(e.ECSTaskRole()),
			},
			NetworkMode: pulumi.StringPtr("awsvpc"),
			Family:      e.CommonNamer.DisplayName(pulumi.String("datadog-agent-ec2")),
			Volumes: classicECS.TaskDefinitionVolumeArray{
				classicECS.TaskDefinitionVolumeArgs{
					HostPath: pulumi.StringPtr("/var/run/docker.sock"),
					Name:     pulumi.String("docker_sock"),
				},
				classicECS.TaskDefinitionVolumeArgs{
					HostPath: pulumi.StringPtr("/proc"),
					Name:     pulumi.String("proc"),
				},
				classicECS.TaskDefinitionVolumeArgs{
					HostPath: pulumi.StringPtr("/sys/fs/cgroup"),
					Name:     pulumi.String("cgroup"),
				},
				classicECS.TaskDefinitionVolumeArgs{
					HostPath: pulumi.StringPtr("/opt/datadog-agent/run"),
					Name:     pulumi.String("dd-logpointdir"),
				},
				classicECS.TaskDefinitionVolumeArgs{
					HostPath: pulumi.StringPtr("/var/run/datadog"),
					Name:     pulumi.String("dd-sockets"),
				},
				classicECS.TaskDefinitionVolumeArgs{
					HostPath: pulumi.StringPtr("/sys/kernel/debug"),
					Name:     pulumi.String("debug"),
				},
			},
		},
	}, e.WithProviders(config.ProviderAWS, config.ProviderAWSX))
}

func ecsLinuxAgentSingleContainerDefinition(e config.CommonEnvironment, apiKeySSMParamName pulumi.StringInput) ecs.TaskDefinitionContainerDefinitionArgs {
	return ecs.TaskDefinitionContainerDefinitionArgs{
		Cpu:       pulumi.IntPtr(100),
		Memory:    pulumi.IntPtr(512),
		Name:      pulumi.StringPtr("datadog-agent"),
		Image:     pulumi.StringPtr(DockerAgentFullImagePath(&e, "public.ecr.aws/datadog/agent")),
		Essential: pulumi.BoolPtr(true),
		LinuxParameters: ecs.TaskDefinitionLinuxParametersArgs{
			Capabilities: ecs.TaskDefinitionKernelCapabilitiesArgs{
				Add: pulumi.ToStringArray([]string{"SYS_ADMIN", "SYS_RESOURCE", "SYS_PTRACE", "NET_ADMIN", "NET_BROADCAST", "NET_RAW", "IPC_LOCK", "CHOWN"}),
			},
		},
		Environment: ecs.TaskDefinitionKeyValuePairArray{
			ecs.TaskDefinitionKeyValuePairArgs{
				Name:  pulumi.StringPtr("DD_DOGSTATSD_SOCKET"),
				Value: pulumi.StringPtr("/var/run/datadog/dsd.socket"),
			},
			ecs.TaskDefinitionKeyValuePairArgs{
				Name:  pulumi.StringPtr("DD_LOGS_ENABLED"),
				Value: pulumi.StringPtr("true"),
			},
			ecs.TaskDefinitionKeyValuePairArgs{
				Name:  pulumi.StringPtr("DD_LOGS_CONFIG_CONTAINER_COLLECT_ALL"),
				Value: pulumi.StringPtr("true"),
			},
			ecs.TaskDefinitionKeyValuePairArgs{
				Name:  pulumi.StringPtr("DD_ECS_COLLECT_RESOURCE_TAGS_EC2"),
				Value: pulumi.StringPtr("true"),
			},
			ecs.TaskDefinitionKeyValuePairArgs{
				Name:  pulumi.StringPtr("DD_DOGSTATSD_NON_LOCAL_TRAFFIC"),
				Value: pulumi.StringPtr("true"),
			},
			ecs.TaskDefinitionKeyValuePairArgs{
				Name:  pulumi.StringPtr("DD_PROCESS_AGENT_ENABLED"),
				Value: pulumi.StringPtr("true"),
			},
		},
		Secrets: ecs.TaskDefinitionSecretArray{
			ecs.TaskDefinitionSecretArgs{
				Name:      pulumi.String("DD_API_KEY"),
				ValueFrom: apiKeySSMParamName,
			},
		},
		MountPoints: ecs.TaskDefinitionMountPointArray{
			ecs.TaskDefinitionMountPointArgs{
				ContainerPath: pulumi.StringPtr("/var/run/docker.sock"),
				SourceVolume:  pulumi.StringPtr("docker_sock"),
				ReadOnly:      pulumi.BoolPtr(true),
			},
			ecs.TaskDefinitionMountPointArgs{
				ContainerPath: pulumi.StringPtr("/host/proc"),
				SourceVolume:  pulumi.StringPtr("proc"),
				ReadOnly:      pulumi.BoolPtr(true),
			},
			ecs.TaskDefinitionMountPointArgs{
				ContainerPath: pulumi.StringPtr("/host/sys/fs/cgroup"),
				SourceVolume:  pulumi.StringPtr("cgroup"),
				ReadOnly:      pulumi.BoolPtr(true),
			},
			ecs.TaskDefinitionMountPointArgs{
				ContainerPath: pulumi.StringPtr("/opt/datadog-agent/run"),
				SourceVolume:  pulumi.StringPtr("dd-logpointdir"),
				ReadOnly:      pulumi.BoolPtr(false),
			},
			ecs.TaskDefinitionMountPointArgs{
				ContainerPath: pulumi.StringPtr("/var/run/datadog"),
				SourceVolume:  pulumi.StringPtr("dd-sockets"),
				ReadOnly:      pulumi.BoolPtr(false),
			},
			ecs.TaskDefinitionMountPointArgs{
				ContainerPath: pulumi.StringPtr("/sys/kernel/debug"),
				SourceVolume:  pulumi.StringPtr("debug"),
				ReadOnly:      pulumi.BoolPtr(false),
			},
		},
		HealthCheck: &ecs.TaskDefinitionHealthCheckArgs{
			Retries:     pulumi.IntPtr(2),
			Command:     pulumi.ToStringArray([]string{"CMD-SHELL", "agent health"}),
			StartPeriod: pulumi.IntPtr(10),
			Interval:    pulumi.IntPtr(30),
			Timeout:     pulumi.IntPtr(5),
		},
		PortMappings: ecs.TaskDefinitionPortMappingArray{
			ecs.TaskDefinitionPortMappingArgs{
				ContainerPort: pulumi.Int(8125),
				HostPort:      pulumi.IntPtr(8125),
				Protocol:      pulumi.StringPtr("udp"),
			},
		},
	}
}
