package ecs

import (
	"github.com/DataDog/test-infra-definitions/aws"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ssm"
	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/ecs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Run(ctx *pulumi.Context) error {
	awsEnv, err := aws.AWSEnvironment(ctx)
	if err != nil {
		return err
	}

	// Create cluster
	ecsCluster, err := CreateEcsCluster(awsEnv, ctx.Stack())
	if err != nil {
		return err
	}

	// Handle capacity providers
	capacityProviders := pulumi.StringArray{}
	if awsEnv.ECSFargateCapacityProvider() {
		capacityProviders = append(capacityProviders, pulumi.String("FARGATE"))
	}

	if awsEnv.ECSLinuxECSOptimizedNodeGroup() {
		cpName, err := NewECSOptimizedNodeGroup(awsEnv, ecsCluster.Name, false)
		if err != nil {
			return err
		}

		capacityProviders = append(capacityProviders, cpName)
	}

	if awsEnv.ECSLinuxECSOptimizedARMNodeGroup() {
		cpName, err := NewECSOptimizedNodeGroup(awsEnv, ecsCluster.Name, true)
		if err != nil {
			return err
		}

		capacityProviders = append(capacityProviders, cpName)
	}

	if awsEnv.ECSLinuxBottlerocketNodeGroup() {
		cpName, err := NewBottlerocketNodeGroup(awsEnv, ecsCluster.Name)
		if err != nil {
			return err
		}

		capacityProviders = append(capacityProviders, cpName)
	}

	if awsEnv.ECSWindowsNodeGroup() {
		cpName, err := NewWindowsNodeGroup(awsEnv, ecsCluster.Name)
		if err != nil {
			return err
		}

		capacityProviders = append(capacityProviders, cpName)
	}

	// Associate capacity providers
	_, err = NewClusterCapacityProvider(awsEnv, ctx.Stack(), ecsCluster.Name, capacityProviders)
	if err != nil {
		return err
	}

	// Create task and service
	if awsEnv.AgentDeploy() {
		apiKeyParam, err := ssm.NewParameter(ctx, ctx.Stack()+"-agent-apikey", &ssm.ParameterArgs{
			Type:  ssm.ParameterTypeSecureString,
			Value: awsEnv.AgentAPIKey(),
		}, pulumi.Provider(awsEnv.Provider))
		if err != nil {
			return err
		}

		testContainer := FargateRedisContainerDefinition(awsEnv, apiKeyParam.Arn)
		taskDef, err := FargateTaskDefinitionWithAgent(awsEnv, ctx.Stack()+"-fg-dd-agent", ctx.Stack()+"-fg-dd-agent", []*ecs.TaskDefinitionContainerDefinitionArgs{testContainer}, apiKeyParam.Name)
		if err != nil {
			return err
		}

		_, err = FargateService(awsEnv, ctx.Stack()+"-fg-dd-agent", ecsCluster.Arn, taskDef.TaskDefinition.Arn())
		if err != nil {
			return err
		}

		ctx.Export("ecs-task-arn", taskDef.TaskDefinition.Arn())
		ctx.Export("ecs-task-family", taskDef.TaskDefinition.Family())
		ctx.Export("ecs-task-version", taskDef.TaskDefinition.Revision())
	}

	ctx.Export("ecs-cluster-name", ecsCluster.Name)
	ctx.Export("ecs-cluster-arn", ecsCluster.Arn)
	return nil
}
