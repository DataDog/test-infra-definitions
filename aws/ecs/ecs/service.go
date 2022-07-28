package ecs

import (
	"reflect"

	"github.com/vboulineau/pulumi-definitions/aws"

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
		TaskDefinition: taskDefArn,
	})
}

func FargateTaskDefinitionWithAgent(ctx *pulumi.Context, environment aws.Environment, family, name string, containers []*ecs.TaskDefinitionContainerDefinitionArgs) (*ecs.FargateTaskDefinition, error) {
	containersMap := make(map[string]ecs.TaskDefinitionContainerDefinitionArgs)
	for _, c := range containers {
		var s string
		// Ugly hack as the implementation of pulumi.StringPtrInput is just `type stringPtr string`
		containersMap[reflect.ValueOf(c.Name).Convert(reflect.PtrTo(s)).String()] = *c
	}
	containersMap["datadog-agent"] = *FargateAgentContainerDefinition(ctx, environment)

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
	})
}

func FargateRedisContainerDefinition(ctx *pulumi.Context, environment aws.Environment) *ecs.TaskDefinitionContainerDefinitionArgs {
	return &ecs.TaskDefinitionContainerDefinitionArgs{
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
	}
}

func FargateAgentContainerDefinition(ctx *pulumi.Context, environment aws.Environment) *ecs.TaskDefinitionContainerDefinitionArgs {
	return &ecs.TaskDefinitionContainerDefinitionArgs{
		Name:      pulumi.StringPtr("datadog-agent"),
		Image:     pulumi.StringPtr("public.ecr.aws/datadog/agent:latest"),
		Essential: pulumi.BoolPtr(true),
		Environment: ecs.TaskDefinitionKeyValuePairArray{
			ecs.TaskDefinitionKeyValuePairArgs{
				Name:  pulumi.StringPtr("ECS_FARGATE"),
				Value: pulumi.StringPtr("true"),
			},
			ecs.TaskDefinitionKeyValuePairArgs{
				Name:  pulumi.StringPtr("DD_DOGSTATSD_SOCKET"),
				Value: pulumi.StringPtr("/var/run/datadog/dsd.socket"),
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
			Retries: pulumi.IntPtr(2),
			Command: pulumi.ToStringArray([]string{"CMD-SHELL", "/probe.sh"}),
		},
		LogConfiguration: getFirelensLogConfiguration("datadog-agent", "datadog-agent", environment.APIKeySSMParamName()),
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
