package ecs

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/datadog/agent"
	"github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	classicECS "github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ecs"
	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/awsx"
	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/ecs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Instance struct {
	pulumi.ResourceState

	Host pulumi.StringOutput
}

func FargateService(e aws.Environment, name string, clusterArn pulumi.StringInput, taskDefArn pulumi.StringInput, lb classicECS.ServiceLoadBalancerArrayInput, opts ...pulumi.ResourceOption) (*ecs.FargateService, error) {
	return ecs.NewFargateService(e.Ctx, e.Namer.ResourceName(name), &ecs.FargateServiceArgs{
		Cluster:      clusterArn,
		Name:         e.CommonNamer.DisplayName(255, pulumi.String(name)),
		DesiredCount: pulumi.IntPtr(1),
		NetworkConfiguration: classicECS.ServiceNetworkConfigurationArgs{
			AssignPublicIp: pulumi.BoolPtr(e.ECSServicePublicIP()),
			SecurityGroups: pulumi.ToStringArray(e.DefaultSecurityGroups()),
			Subnets:        e.RandomSubnets(),
		},
		LoadBalancers:        lb,
		TaskDefinition:       taskDefArn,
		EnableExecuteCommand: pulumi.BoolPtr(true),
	}, utils.MergeOptions(opts, e.WithProviders(config.ProviderAWS, config.ProviderAWSX))...)
}

func FargateTaskDefinitionWithAgent(
	e aws.Environment,
	name string,
	family pulumi.StringInput,
	cpu, memory int,
	containers map[string]ecs.TaskDefinitionContainerDefinitionArgs,
	apiKeySSMParamName pulumi.StringInput,
	fakeintake *fakeintake.Fakeintake,
	opts ...pulumi.ResourceOption,
) (*ecs.FargateTaskDefinition, error) {
	containers["datadog-agent"] = *agent.ECSFargateLinuxContainerDefinition(*e.CommonEnvironment, "public.ecr.aws/datadog/agent:latest", apiKeySSMParamName, fakeintake, GetFirelensLogConfiguration(pulumi.String("datadog-agent"), pulumi.String("datadog-agent"), apiKeySSMParamName))
	containers["log_router"] = *FargateFirelensContainerDefinition()

	return ecs.NewFargateTaskDefinition(e.Ctx, e.Namer.ResourceName(name), &ecs.FargateTaskDefinitionArgs{
		Containers: containers,
		Cpu:        pulumi.StringPtr(fmt.Sprintf("%d", cpu)),
		Memory:     pulumi.StringPtr(fmt.Sprintf("%d", memory)),
		ExecutionRole: &awsx.DefaultRoleWithPolicyArgs{
			RoleArn: pulumi.StringPtr(e.ECSTaskExecutionRole()),
		},
		TaskRole: &awsx.DefaultRoleWithPolicyArgs{
			RoleArn: pulumi.StringPtr(e.ECSTaskRole()),
		},
		Family: e.CommonNamer.DisplayName(255, family),
		Volumes: classicECS.TaskDefinitionVolumeArray{
			classicECS.TaskDefinitionVolumeArgs{
				Name: pulumi.String("dd-sockets"),
			},
		},
	}, utils.MergeOptions(opts, e.WithProviders(config.ProviderAWS, config.ProviderAWSX))...)
}

func FargateRedisContainerDefinition(apiKeySSMParamName pulumi.StringInput) *ecs.TaskDefinitionContainerDefinitionArgs {
	return &ecs.TaskDefinitionContainerDefinitionArgs{
		Cpu:       pulumi.IntPtr(0),
		Name:      pulumi.String("redis"),
		Image:     pulumi.String("redis:latest"),
		Essential: pulumi.BoolPtr(true),
		DependsOn: ecs.TaskDefinitionContainerDependencyArray{
			ecs.TaskDefinitionContainerDependencyArgs{
				ContainerName: pulumi.String("datadog-agent"),
				Condition:     pulumi.String("HEALTHY"),
			},
		},
		LogConfiguration: GetFirelensLogConfiguration(pulumi.String("redis"), pulumi.String("redis"), apiKeySSMParamName),
		MountPoints:      ecs.TaskDefinitionMountPointArray{},
		Environment:      ecs.TaskDefinitionKeyValuePairArray{},
		PortMappings:     ecs.TaskDefinitionPortMappingArray{},
		VolumesFrom:      ecs.TaskDefinitionVolumeFromArray{},
	}
}

func FargateFirelensContainerDefinition() *ecs.TaskDefinitionContainerDefinitionArgs {
	return &ecs.TaskDefinitionContainerDefinitionArgs{
		Cpu:       pulumi.IntPtr(0),
		User:      pulumi.StringPtr("0"),
		Name:      pulumi.String("log_router"),
		Image:     pulumi.String("amazon/aws-for-fluent-bit:stable"),
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

func GetFirelensLogConfiguration(source, service, apiKeyParamName pulumi.StringInput) ecs.TaskDefinitionLogConfigurationPtrInput {
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
