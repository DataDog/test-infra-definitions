package ecs

import (
	"github.com/DataDog/test-infra-definitions/aws"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ssm"
	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/ecs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Run(ctx *pulumi.Context) error {
	awsEnv := aws.AWSEnvironment(ctx)

	capacityProviders := pulumi.StringArray{}
	// Handle capacity providers
	if awsEnv.ECSFargateCapacityProvider() {
		capacityProviders = append(capacityProviders, pulumi.String("FARGATE"))
	}

	if awsEnv.ECSLinuxECSOptimizedNodeGroup() {
		cpName, err := NewECSOptimizedNodeGroup(ctx, awsEnv)
		if err != nil {
			return err
		}

		capacityProviders = append(capacityProviders, cpName)
	}

	// Create cluster
	ecsCluster, err := CreateEcsCluster(ctx, awsEnv, capacityProviders)
	if err != nil {
		return err
	}

	// Create task and service
	if awsEnv.DeployAgent() {
		apiKeyParam, err := ssm.NewParameter(ctx, "agent-api-key", &ssm.ParameterArgs{
			Type:  ssm.ParameterTypeSecureString,
			Value: awsEnv.AgentAPIKey(),
		})
		if err != nil {
			return err
		}

		testContainer := FargateRedisContainerDefinition(ctx, awsEnv, apiKeyParam.Arn)
		taskDef, err := FargateTaskDefinitionWithAgent(ctx, awsEnv, "ci-tasks", ctx.Stack(), []*ecs.TaskDefinitionContainerDefinitionArgs{testContainer}, apiKeyParam.Name)
		if err != nil {
			return err
		}

		_, err = FargateService(ctx, awsEnv, ctx.Stack(), ecsCluster.Arn, taskDef.TaskDefinition.Arn())
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
