package agent

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	ddfakeintake "github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"

	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/ecs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func ECSFargateLinuxContainerDefinition(e config.CommonEnvironment, apiKeySSMParamName pulumi.StringInput, fakeintake *ddfakeintake.ConnectionExporter, logConfig ecs.TaskDefinitionLogConfigurationPtrInput) *ecs.TaskDefinitionContainerDefinitionArgs {
	fakeintakeEnv := []ecs.TaskDefinitionKeyValuePairInput{}
	if fakeintake != nil {
		fakeintakeEnv = []ecs.TaskDefinitionKeyValuePairInput{
			ecs.TaskDefinitionKeyValuePairArgs{
				Name:  pulumi.StringPtr("DD_ADDITIONAL_ENDPOINTS"),
				Value: pulumi.Sprintf(`{"http://%s": ["FAKEAPIKEY"]}`, fakeintake.Host),
			},
			ecs.TaskDefinitionKeyValuePairArgs{
				Name:  pulumi.String("DD_LOGS_CONFIG_ADDITIONAL_ENDPOINTS"),
				Value: pulumi.Sprintf(`[{"host": "%s", "port": 80, "is_reliable": true, "usessl": false}]`, fakeintake.Host),
			},
		}
	}

	return &ecs.TaskDefinitionContainerDefinitionArgs{
		Cpu:       pulumi.IntPtr(0),
		Name:      pulumi.StringPtr("datadog-agent"),
		Image:     pulumi.Sprintf(DockerAgentFullImagePath(&e, "public.ecr.aws/datadog/agent")),
		Essential: pulumi.BoolPtr(true),
		Environment: append(ecs.TaskDefinitionKeyValuePairArray{
			ecs.TaskDefinitionKeyValuePairArgs{
				Name:  pulumi.StringPtr("DD_DOGSTATSD_SOCKET"),
				Value: pulumi.StringPtr("/var/run/datadog/dsd.socket"),
			},
			ecs.TaskDefinitionKeyValuePairArgs{
				Name:  pulumi.StringPtr("ECS_FARGATE"),
				Value: pulumi.StringPtr("true"),
			},
		}, fakeintakeEnv...),
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
