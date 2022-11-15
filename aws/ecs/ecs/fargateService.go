package ecs

import (
	"reflect"

	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/datadog/agent"

	classicECS "github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ecs"
	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/awsx"
	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/ecs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func FargateService(e aws.Environment, name string, clusterArn pulumi.StringInput, taskDefArn pulumi.StringInput) (*ecs.FargateService, error) {
	return ecs.NewFargateService(e.Ctx, name, &ecs.FargateServiceArgs{
		Cluster:      clusterArn,
		Name:         pulumi.StringPtr(name),
		DesiredCount: pulumi.IntPtr(1),
		NetworkConfiguration: classicECS.ServiceNetworkConfigurationArgs{
			AssignPublicIp: pulumi.BoolPtr(e.ECSServicePublicIP()),
			SecurityGroups: pulumi.ToStringArray(e.DefaultSecurityGroups()),
			Subnets:        pulumi.ToStringArray(e.DefaultSubnets()),
		},
		TaskDefinition:            taskDefArn,
		EnableExecuteCommand:      pulumi.BoolPtr(true),
		ContinueBeforeSteadyState: pulumi.BoolPtr(true),
	}, pulumi.Provider(e.Provider))
}

func FargateTaskDefinitionWithAgent(e aws.Environment, family, name string, containers []*ecs.TaskDefinitionContainerDefinitionArgs, apiKeySSMParamName pulumi.StringInput) (*ecs.FargateTaskDefinition, error) {
	containersMap := make(map[string]ecs.TaskDefinitionContainerDefinitionArgs)
	for _, c := range containers {
		// Ugly hack as the implementation of pulumi.StringPtrInput is just `type stringPtr string`
		containersMap[reflect.ValueOf(c.Name).Elem().String()] = *c
	}
	containersMap["datadog-agent"] = *agent.ECSFargateLinuxContainerDefinition(*e.CommonEnvironment, apiKeySSMParamName, getFirelensLogConfiguration(pulumi.String("datadog-agent"), pulumi.String("datadog-agent"), apiKeySSMParamName))
	containersMap["log_router"] = *FargateFirelensContainerDefinition(e)

	return ecs.NewFargateTaskDefinition(e.Ctx, name, &ecs.FargateTaskDefinitionArgs{
		Containers: containersMap,
		Cpu:        pulumi.StringPtr("1024"),
		Memory:     pulumi.StringPtr("2048"),
		ExecutionRole: &awsx.DefaultRoleWithPolicyArgs{
			RoleArn: pulumi.StringPtr(e.ECSTaskExecutionRole()),
		},
		TaskRole: &awsx.DefaultRoleWithPolicyArgs{
			RoleArn: pulumi.StringPtr(e.ECSTaskRole()),
		},
		Family: pulumi.StringPtr(family),
		Volumes: classicECS.TaskDefinitionVolumeArray{
			classicECS.TaskDefinitionVolumeArgs{
				Name: pulumi.String("dd-sockets"),
			},
		},
	}, pulumi.Provider(e.Provider))
}

func FargateRedisContainerDefinition(e aws.Environment, apiKeySSMParamName pulumi.StringInput) *ecs.TaskDefinitionContainerDefinitionArgs {
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
		LogConfiguration: getFirelensLogConfiguration(pulumi.String("redis"), pulumi.String("redis"), apiKeySSMParamName),
		MountPoints:      ecs.TaskDefinitionMountPointArray{},
		Environment:      ecs.TaskDefinitionKeyValuePairArray{},
		PortMappings:     ecs.TaskDefinitionPortMappingArray{},
		VolumesFrom:      ecs.TaskDefinitionVolumeFromArray{},
	}
}

func FargateFirelensContainerDefinition(e aws.Environment) *ecs.TaskDefinitionContainerDefinitionArgs {
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

func getFirelensLogConfiguration(source, service, apiKeyParamName pulumi.StringInput) ecs.TaskDefinitionLogConfigurationPtrInput {
	return ecs.TaskDefinitionLogConfigurationArgs{
		LogDriver: pulumi.String("awsfirelens"),
		Options: pulumi.StringMap{
			"Name":           pulumi.String("datadog"),
			"Host":           pulumi.String("http-intake.logs.datadoghq.com"),
			"dd_service":     service,
			"dd_source":      source,
			"dd_message_key": pulumi.String("log"),
			"TLS":            pulumi.String("on"),
			"provider":       pulumi.String("ecs"),
		},
		SecretOptions: ecs.TaskDefinitionSecretArray{
			ecs.TaskDefinitionSecretArgs{
				Name:      pulumi.String("apikey"),
				ValueFrom: apiKeyParamName,
			},
		},
	}
}
