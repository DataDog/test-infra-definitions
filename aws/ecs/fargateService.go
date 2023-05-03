package ecs

import (
	"fmt"
	"reflect"

	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/datadog/agent"
	ddfakeintake "github.com/DataDog/test-infra-definitions/datadog/fakeintake"
	"github.com/cenkalti/backoff/v4"

	classicECS "github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ecs"
	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/awsx"
	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/ecs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	oneSecond     = 1000
	sleepInterval = 1 * oneSecond
	maxRetries    = 120
)

func FargateService(e aws.Environment, name string, clusterArn pulumi.StringInput, taskDefArn pulumi.StringInput) (*ecs.FargateService, error) {
	return ecs.NewFargateService(e.Ctx, e.Namer.ResourceName(name), &ecs.FargateServiceArgs{
		Cluster:      clusterArn,
		Name:         e.CommonNamer.DisplayName(pulumi.String(name)),
		DesiredCount: pulumi.IntPtr(1),
		NetworkConfiguration: classicECS.ServiceNetworkConfigurationArgs{
			AssignPublicIp: pulumi.BoolPtr(e.ECSServicePublicIP()),
			SecurityGroups: pulumi.ToStringArray(e.DefaultSecurityGroups()),
			Subnets:        pulumi.ToStringArray(e.DefaultSubnets()),
		},
		TaskDefinition:            taskDefArn,
		EnableExecuteCommand:      pulumi.BoolPtr(true),
		ContinueBeforeSteadyState: pulumi.BoolPtr(true),
	}, e.ResourceProvidersOption())
}

func FargateTaskDefinitionWithAgent(e aws.Environment, name string, family pulumi.StringInput, containers []*ecs.TaskDefinitionContainerDefinitionArgs, apiKeySSMParamName pulumi.StringInput) (*ecs.FargateTaskDefinition, error) {
	containersMap := make(map[string]ecs.TaskDefinitionContainerDefinitionArgs)
	for _, c := range containers {
		// Ugly hack as the implementation of pulumi.StringPtrInput is just `type stringPtr string`
		containersMap[reflect.ValueOf(c.Name).Elem().String()] = *c
	}
	containersMap["datadog-agent"] = *agent.ECSFargateLinuxContainerDefinition(*e.CommonEnvironment, apiKeySSMParamName, getFirelensLogConfiguration(pulumi.String("datadog-agent"), pulumi.String("datadog-agent"), apiKeySSMParamName))
	containersMap["log_router"] = *FargateFirelensContainerDefinition()

	return ecs.NewFargateTaskDefinition(e.Ctx, e.Namer.ResourceName(name), &ecs.FargateTaskDefinitionArgs{
		Containers: containersMap,
		Cpu:        pulumi.StringPtr("1024"),
		Memory:     pulumi.StringPtr("2048"),
		ExecutionRole: &awsx.DefaultRoleWithPolicyArgs{
			RoleArn: pulumi.StringPtr(e.ECSTaskExecutionRole()),
		},
		TaskRole: &awsx.DefaultRoleWithPolicyArgs{
			RoleArn: pulumi.StringPtr(e.ECSTaskRole()),
		},
		Family: e.CommonNamer.DisplayName(family),
		Volumes: classicECS.TaskDefinitionVolumeArray{
			classicECS.TaskDefinitionVolumeArgs{
				Name: pulumi.String("dd-sockets"),
			},
		},
	}, e.ResourceProvidersOption())
}

func FargateRedisContainerDefinition(apiKeySSMParamName pulumi.StringInput) *ecs.TaskDefinitionContainerDefinitionArgs {
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

func FargateFirelensContainerDefinition() *ecs.TaskDefinitionContainerDefinitionArgs {
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

// FargateServiceFakeintake deploys one fakeintake container to a dedicated Fargate cluster
// Hardcoded on sandbox
// TODO add new aws dedicated accounts once ready
func FargateServiceFakeintake(e aws.Environment) (ipAddress pulumi.StringOutput, err error) {
	taskDef, err := ddfakeintake.FargateLinuxTaskDefinition(e, e.Namer.ResourceName("fakeintake-taskdef"))
	if err != nil {
		return pulumi.StringOutput{}, err
	}
	fargateService, err := FargateService(e, e.Namer.ResourceName("fakeintake-srv"), pulumi.String(e.ECSFargateFakeintakeClusterArn()), taskDef.TaskDefinition.Arn())
	// Hack passing taskDef.TaskDefinition.Arn() to execute apply function
	// when taskDef has an ARN, thus it is defined on AWS side
	ipAddress = pulumi.All(taskDef.TaskDefinition.Arn(), fargateService.Service.Name()).ApplyT(func(args []any) (string, error) {
		var ipAddress string
		err := backoff.Retry(func() error {
			fmt.Println("waiting for fakeintake task private ip")
			serviceName := args[1].(string)
			ecsClient, err := NewECSClient(e.Ctx.Context(), e.Region())
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
