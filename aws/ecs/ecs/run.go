package ecs

import (
	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/datadog/agent"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ssm"
	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/ecs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Run(ctx *pulumi.Context) error {
	awsEnv, err := aws.NewEnvironment(ctx)
	if err != nil {
		return err
	}

	// Create cluster
	ecsCluster, err := CreateEcsCluster(awsEnv, "ecs")
	if err != nil {
		return err
	}

	// Handle capacity providers
	capacityProviders := pulumi.StringArray{}
	if awsEnv.ECSFargateCapacityProvider() {
		capacityProviders = append(capacityProviders, pulumi.String("FARGATE"))
	}

	linuxNodeGroupPresent := false
	if awsEnv.ECSLinuxECSOptimizedNodeGroup() {
		cpName, err := NewECSOptimizedNodeGroup(awsEnv, ecsCluster.Name, false)
		if err != nil {
			return err
		}

		capacityProviders = append(capacityProviders, cpName)
		linuxNodeGroupPresent = true
	}

	if awsEnv.ECSLinuxECSOptimizedARMNodeGroup() {
		cpName, err := NewECSOptimizedNodeGroup(awsEnv, ecsCluster.Name, true)
		if err != nil {
			return err
		}

		capacityProviders = append(capacityProviders, cpName)
		linuxNodeGroupPresent = true
	}

	if awsEnv.ECSLinuxBottlerocketNodeGroup() {
		cpName, err := NewBottlerocketNodeGroup(awsEnv, ecsCluster.Name)
		if err != nil {
			return err
		}

		capacityProviders = append(capacityProviders, cpName)
		linuxNodeGroupPresent = true
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
		apiKeyParam, err := ssm.NewParameter(ctx, awsEnv.Namer.ResourceName("agent-apikey"), &ssm.ParameterArgs{
			Name:  awsEnv.CommonNamer.DisplayName(pulumi.String("agent-apikey")),
			Type:  ssm.ParameterTypeSecureString,
			Value: awsEnv.AgentAPIKey(),
		}, awsEnv.ResourceProvidersOption())
		if err != nil {
			return err
		}

		// Deploy Fargate Agent
		testContainer := FargateRedisContainerDefinition(awsEnv, apiKeyParam.Arn)
		taskDef, err := FargateTaskDefinitionWithAgent(awsEnv, "fg-datadog-agent", pulumi.String("fg-datadog-agent"), []*ecs.TaskDefinitionContainerDefinitionArgs{testContainer}, apiKeyParam.Name)
		if err != nil {
			return err
		}

		_, err = FargateService(awsEnv, "fg-datadog-agent", ecsCluster.Arn, taskDef.TaskDefinition.Arn())
		if err != nil {
			return err
		}

		ctx.Export("agent-fargate-task-arn", taskDef.TaskDefinition.Arn())
		ctx.Export("agent-fargate-task-family", taskDef.TaskDefinition.Family())
		ctx.Export("agent-fargate-task-version", taskDef.TaskDefinition.Revision())

		// Deploy EC2 Agent
		if linuxNodeGroupPresent {
			agentDaemon, err := agent.ECSLinuxDaemonDefinition(awsEnv, "ec2-linux-dd-agent", apiKeyParam.Name, ecsCluster.Arn)
			if err != nil {
				return err
			}

			ctx.Export("agent-ec2-linux-task-arn", agentDaemon.TaskDefinition.Arn())
			ctx.Export("agent-ec2-linux-task-family", agentDaemon.TaskDefinition.Family())
			ctx.Export("agent-ec2-linux-task-version", agentDaemon.TaskDefinition.Revision())
		}
	}

	ctx.Export("ecs-cluster-name", ecsCluster.Name)
	ctx.Export("ecs-cluster-arn", ecsCluster.Arn)
	return nil
}
