package ecs

import (
	"reflect"

	"github.com/DataDog/test-infra-definitions/aws"

	classicECS "github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ecs"
	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/awsx"
	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/ecs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func FargateService(ctx *pulumi.Context, environment aws.Environment, name string, clusterArn pulumi.StringInput, taskDefArn pulumi.StringInput) (*ecs.FargateService, error) {
	return ecs.NewFargateService(ctx, name, &ecs.FargateServiceArgs{
		Cluster:      clusterArn,
		Name:         pulumi.StringPtr(name),
		DesiredCount: pulumi.IntPtr(1),
		NetworkConfiguration: classicECS.ServiceNetworkConfigurationArgs{
			AssignPublicIp: pulumi.BoolPtr(environment.AssignPublicIP()),
			SecurityGroups: pulumi.ToStringArray(environment.DefaultSecurityGroups()),
			Subnets:        pulumi.ToStringArray([]string{environment.DefaultSubnet()}),
		},
		TaskDefinition:            taskDefArn,
		EnableExecuteCommand:      pulumi.BoolPtr(true),
		ContinueBeforeSteadyState: pulumi.BoolPtr(true),
	})
}

func FargateTaskDefinitionWithAgent(ctx *pulumi.Context, environment aws.Environment, family, name string, containers []*ecs.TaskDefinitionContainerDefinitionArgs) (*ecs.FargateTaskDefinition, error) {
	containersMap := make(map[string]ecs.TaskDefinitionContainerDefinitionArgs)
	for _, c := range containers {
		// Ugly hack as the implementation of pulumi.StringPtrInput is just `type stringPtr string`
		containersMap[reflect.ValueOf(c.Name).Elem().String()] = *c
	}
	containersMap["datadog-agent"] = *FargateAgentContainerDefinition(ctx, environment)
	containersMap["log_router"] = *FargateFirelensContainerDefinition(ctx, environment)

	return ecs.NewFargateTaskDefinition(ctx, name, &ecs.FargateTaskDefinitionArgs{
		Containers: containersMap,
		Cpu:        pulumi.StringPtr("1024"),
		Memory:     pulumi.StringPtr("2048"),
		ExecutionRole: &awsx.DefaultRoleWithPolicyArgs{
			RoleArn: pulumi.StringPtr(environment.ECSTaskExecutionRole()),
		},
		TaskRole: &awsx.DefaultRoleWithPolicyArgs{
			RoleArn: pulumi.StringPtr(environment.ECSTaskRole()),
		},
		Family: pulumi.StringPtr(family),
		Volumes: classicECS.TaskDefinitionVolumeArray{
			classicECS.TaskDefinitionVolumeArgs{
				Name: pulumi.String("dd-sockets"),
			},
		},
	})
}

func FargateRedisContainerDefinition(ctx *pulumi.Context, environment aws.Environment) *ecs.TaskDefinitionContainerDefinitionArgs {
	return &ecs.TaskDefinitionContainerDefinitionArgs{
		Cpu:       pulumi.IntPtr(0),
		Name:      pulumi.StringPtr("redis"),
		Image:     pulumi.StringPtr("redis:latest"),
		Essential: pulumi.BoolPtr(true),
		DependsOn: ecs.TaskDefinitionContainerDependencyArray{
			ecs.TaskDefinitionContainerDependencyArgs{
				ContainerName: pulumi.String("datadog-agent"),
				Condition:     pulumi.String("HEALTHY"),
			},
		},
		LogConfiguration: getFirelensLogConfiguration("redis", "redis", environment.APIKeySSMParamName()),
		MountPoints:      ecs.TaskDefinitionMountPointArray{},
		Environment:      ecs.TaskDefinitionKeyValuePairArray{},
		PortMappings:     ecs.TaskDefinitionPortMappingArray{},
		VolumesFrom:      ecs.TaskDefinitionVolumeFromArray{},
	}
}

func FargateAgentContainerDefinition(ctx *pulumi.Context, environment aws.Environment) *ecs.TaskDefinitionContainerDefinitionArgs {
	return &ecs.TaskDefinitionContainerDefinitionArgs{
		Cpu:       pulumi.IntPtr(0),
		Name:      pulumi.StringPtr("datadog-agent"),
		Image:     pulumi.StringPtr("public.ecr.aws/datadog/agent:latest"),
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
				ValueFrom: pulumi.String(environment.APIKeySSMParamName()),
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
		LogConfiguration: getFirelensLogConfiguration("datadog-agent", "datadog-agent", environment.APIKeySSMParamName()),
		PortMappings:     ecs.TaskDefinitionPortMappingArray{},
		VolumesFrom:      ecs.TaskDefinitionVolumeFromArray{},
	}
}

func FargateFirelensContainerDefinition(ctx *pulumi.Context, environment aws.Environment) *ecs.TaskDefinitionContainerDefinitionArgs {
	return &ecs.TaskDefinitionContainerDefinitionArgs{
		Cpu:       pulumi.IntPtr(0),
		User:      pulumi.StringPtr("0"),
		Name:      pulumi.StringPtr("log_router"),
		Image:     pulumi.StringPtr("amazon/aws-for-fluent-bit:latest"),
		Essential: pulumi.BoolPtr(true),
		FirelensConfiguration: ecs.TaskDefinitionFirelensConfigurationArgs{
			Type: pulumi.String("fluentbit"),
			Options: pulumi.StringMap{
				"enable-ecs-log-metadata": pulumi.String("true"),
			},
		},
		MountPoints:  ecs.TaskDefinitionMountPointArray{},
		Environment:  ecs.TaskDefinitionKeyValuePairArray{},
		PortMappings: ecs.TaskDefinitionPortMappingArray{},
		VolumesFrom:  ecs.TaskDefinitionVolumeFromArray{},
	}
}

func getFirelensLogConfiguration(source, service, apiKeyParamName string) ecs.TaskDefinitionLogConfigurationPtrInput {
	return ecs.TaskDefinitionLogConfigurationArgs{
		LogDriver: pulumi.String("awsfirelens"),
		Options: pulumi.StringMap{
			"Name":           pulumi.String("datadog"),
			"Host":           pulumi.String("http-intake.logs.datadoghq.com"),
			"dd_service":     pulumi.String(service),
			"dd_source":      pulumi.String(source),
			"dd_message_key": pulumi.String("log"),
			"TLS":            pulumi.String("on"),
			"provider":       pulumi.String("ecs"),
		},
		SecretOptions: ecs.TaskDefinitionSecretArray{
			ecs.TaskDefinitionSecretArgs{
				Name:      pulumi.String("apikey"),
				ValueFrom: pulumi.String(apiKeyParamName),
			},
		},
	}
}
