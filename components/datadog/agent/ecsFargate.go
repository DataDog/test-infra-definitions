package agent

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"

	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/ecs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func ECSFargateLinuxContainerDefinition(e config.CommonEnvironment, image string, apiKeySSMParamName pulumi.StringInput, fakeintake *fakeintake.Fakeintake, logConfig ecs.TaskDefinitionLogConfigurationPtrInput) *ecs.TaskDefinitionContainerDefinitionArgs {
	if image == "" {
		image = dockerAgentFullImagePath(&e, "public.ecr.aws/datadog/agent", "latest")
	}

	return &ecs.TaskDefinitionContainerDefinitionArgs{
		Cpu:       pulumi.IntPtr(0),
		Name:      pulumi.String("datadog-agent"),
		Image:     pulumi.String(image),
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
			ecs.TaskDefinitionKeyValuePairArgs{
				Name:  pulumi.StringPtr("DD_CHECKS_TAG_CARDINALITY"),
				Value: pulumi.StringPtr("high"),
			},
		}, ecsFakeintakeAdditionalEndpointsEnv(fakeintake)...),
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
		DockerLabels: pulumi.StringMap{
			"com.datadoghq.ad.checks": pulumi.String(utils.JSONMustMarshal(
				map[string]interface{}{
					"openmetrics": map[string]interface{}{
						"init_configs": []map[string]interface{}{},
						"instances": []map[string]interface{}{
							{
								"openmetrics_endpoint": "http://localhost:5000/telemetry",
								"namespace":            "datadog.agent",
								"metrics": []string{
									".*",
								},
							},
						},
					},
				},
			)),
		},
	}
}
