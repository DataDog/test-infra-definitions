package ecs

import (
	"errors"

	"github.com/vboulineau/pulumi-definitions/aws"
	"github.com/vboulineau/pulumi-definitions/common/config"

	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/ecs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Run(ctx *pulumi.Context, env config.Environment) error {
	awsEnv, ok := env.(aws.Environment)
	if !ok {
		return errors.New("creating ECS Cluster is only supported on AWS Environments")
	}

	// Create cluster
	ecsCluster, err := CreateEcsCluster(ctx, awsEnv)
	if err != nil {
		return err
	}

	// Create task and service
	testContainer := FargateRedisContainerDefinition(ctx, awsEnv)
	taskDef, err := FargateTaskDefinitionWithAgent(ctx, awsEnv, "ci-tasks", ctx.Stack(), []*ecs.TaskDefinitionContainerDefinitionArgs{testContainer})
	if err != nil {
		return err
	}

	_, err = FargateService(ctx, awsEnv, ctx.Stack(), ecsCluster.Arn, taskDef.TaskDefinition.Arn())
	if err != nil {
		return err
	}

	ctx.Export("ecs-cluster-name", ecsCluster.Name)
	ctx.Export("ecs-cluster-arn", ecsCluster.Arn)
	ctx.Export("ecs-task-arn", taskDef.TaskDefinition.Arn())
	ctx.Export("ecs-task-family", taskDef.TaskDefinition.Family())
	ctx.Export("ecs-task-version", taskDef.TaskDefinition.Revision())

	return nil
}
