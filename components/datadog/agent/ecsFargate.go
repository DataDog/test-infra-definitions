package agent

import (
	"github.com/DataDog/test-infra-definitions/common/config"

	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/ecs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func ECSFargateLinuxContainerDefinition(e config.CommonEnvironment, apiKeySSMParamName pulumi.StringInput, logConfig ecs.TaskDefinitionLogConfigurationPtrInput) *ecs.TaskDefinitionContainerDefinitionArgs {
	return &ecs.TaskDefinitionContainerDefinitionArgs{
		Cpu:       pulumi.IntPtr(0),
		Name:      pulumi.StringPtr("datadog-agent"),
		Image:     pulumi.Sprintf(DockerFullImagePath(&e, "public.ecr.aws/datadog/agent")),
		Essential: pulumi.BoolPtr(true),
		Environment: ecs.TaskDefinitionKeyValuePairArray{
			ecs.TaskDefinitionKeyValuePairArgs{
				Name:  pulumi.StringPtr("DD_DOGSTATSD_SOCKET"),
				Value: pulumi.StringPtr("/var/run/datadog/dsd.socket"),
			},
			ecs.TaskDefinitionKeyValuePairArgs{
				Name:  pulumi.StringPtr("ECS_FARGATE"),
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
				ContainerPath: pulumi.StringPtr("/var/run/datadog"),
				SourceVolume:  pulumi.StringPtr("dd-sockets"),
			},
		},
		HealthCheck: &ecs.TaskDefinitionHealthCheckArgs{
			Retries:     pulumi.IntPtr(2),
			Command:     pulumi.ToStringArray([]string{"CMD-SHELL", "/probe.sh"}),
			StartPeriod: pulumi.IntPtr(10),
			Interval:    pulumi.IntPtr(30),
			Timeout:     pulumi.IntPtr(5),
		},
		LogConfiguration: logConfig,
		PortMappings:     ecs.TaskDefinitionPortMappingArray{},
		VolumesFrom:      ecs.TaskDefinitionVolumeFromArray{},
	}
}
