package ecs

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/components/datadog/agent"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/cpustress"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/dogstatsd"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/nginx"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/prometheus"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/redis"
	fakeintakeComp "github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
	resourcesAws "github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/resources/aws/ecs"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/fakeintake"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ssm"
	ecsx "github.com/pulumi/pulumi-awsx/sdk/go/awsx/ecs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Run(ctx *pulumi.Context) error {
	awsEnv, err := resourcesAws.NewEnvironment(ctx)
	if err != nil {
		return err
	}

	// Create cluster
	ecsCluster, err := ecs.CreateEcsCluster(awsEnv, "ecs")
	if err != nil {
		return err
	}

	// Export cluster’s properties
	ctx.Export("ecs-cluster-name", ecsCluster.Name)
	ctx.Export("ecs-cluster-arn", ecsCluster.Arn)

	// Handle capacity providers
	capacityProviders := pulumi.StringArray{}
	if awsEnv.ECSFargateCapacityProvider() {
		capacityProviders = append(capacityProviders, pulumi.String("FARGATE"))
	}

	linuxNodeGroupPresent := false
	if awsEnv.ECSLinuxECSOptimizedNodeGroup() {
		cpName, err := ecs.NewECSOptimizedNodeGroup(awsEnv, ecsCluster.Name, false)
		if err != nil {
			return err
		}

		capacityProviders = append(capacityProviders, cpName)
		linuxNodeGroupPresent = true
	}

	if awsEnv.ECSLinuxECSOptimizedARMNodeGroup() {
		cpName, err := ecs.NewECSOptimizedNodeGroup(awsEnv, ecsCluster.Name, true)
		if err != nil {
			return err
		}

		capacityProviders = append(capacityProviders, cpName)
		linuxNodeGroupPresent = true
	}

	if awsEnv.ECSLinuxBottlerocketNodeGroup() {
		cpName, err := ecs.NewBottlerocketNodeGroup(awsEnv, ecsCluster.Name)
		if err != nil {
			return err
		}

		capacityProviders = append(capacityProviders, cpName)
		linuxNodeGroupPresent = true
	}

	if awsEnv.ECSWindowsNodeGroup() {
		cpName, err := ecs.NewWindowsNodeGroup(awsEnv, ecsCluster.Name)
		if err != nil {
			return err
		}

		capacityProviders = append(capacityProviders, cpName)
	}

	// Associate capacity providers
	_, err = ecs.NewClusterCapacityProvider(awsEnv, ctx.Stack(), ecsCluster.Name, capacityProviders)
	if err != nil {
		return err
	}

	// Create task and service
	if awsEnv.AgentDeploy() {
		var fakeIntake *fakeintakeComp.Fakeintake
		if awsEnv.GetCommonEnvironment().AgentUseFakeintake() {
			if fakeIntake, err = fakeintake.NewECSFargateInstance(awsEnv, "ecs"); err != nil {
				return err
			}
		}
		apiKeyParam, err := ssm.NewParameter(ctx, awsEnv.Namer.ResourceName("agent-apikey"), &ssm.ParameterArgs{
			Name:  awsEnv.CommonNamer.DisplayName(1011, pulumi.String("agent-apikey")),
			Type:  ssm.ParameterTypeSecureString,
			Value: awsEnv.AgentAPIKey(),
		}, awsEnv.WithProviders(config.ProviderAWS))
		if err != nil {
			return err
		}

		// Deploy Fargate Agent
		testContainer := ecs.FargateRedisContainerDefinition(apiKeyParam.Arn)
		taskDef, err := ecs.FargateTaskDefinitionWithAgent(awsEnv, "fg-datadog-agent", pulumi.String("fg-datadog-agent"), 1024, 2048, map[string]ecsx.TaskDefinitionContainerDefinitionArgs{"redis": *testContainer}, apiKeyParam.Name, fakeIntake)
		if err != nil {
			return err
		}

		_, err = ecs.FargateService(awsEnv, "fg-datadog-agent", ecsCluster.Arn, taskDef.TaskDefinition.Arn(), nil)
		if err != nil {
			return err
		}

		ctx.Export("agent-fargate-task-arn", taskDef.TaskDefinition.Arn())
		ctx.Export("agent-fargate-task-family", taskDef.TaskDefinition.Family())
		ctx.Export("agent-fargate-task-version", taskDef.TaskDefinition.Revision())

		// Deploy EC2 Agent
		if linuxNodeGroupPresent {
			agentDaemon, err := agent.ECSLinuxDaemonDefinition(awsEnv, "ec2-linux-dd-agent", apiKeyParam.Name, fakeIntake, ecsCluster.Arn)
			if err != nil {
				return err
			}

			ctx.Export("agent-ec2-linux-task-arn", agentDaemon.TaskDefinition.Arn())
			ctx.Export("agent-ec2-linux-task-family", agentDaemon.TaskDefinition.Family())
			ctx.Export("agent-ec2-linux-task-version", agentDaemon.TaskDefinition.Revision())
		}
	}

	// Deploy testing workload
	if awsEnv.TestingWorkloadDeploy() {
		if _, err := nginx.EcsAppDefinition(awsEnv, ecsCluster.Arn); err != nil {
			return err
		}

		if _, err := redis.EcsAppDefinition(awsEnv, ecsCluster.Arn); err != nil {
			return err
		}

		if _, err := cpustress.EcsAppDefinition(awsEnv, ecsCluster.Arn); err != nil {
			return err
		}

		if _, err := dogstatsd.EcsAppDefinition(awsEnv, ecsCluster.Arn); err != nil {
			return err
		}

		if _, err := prometheus.EcsAppDefinition(awsEnv, ecsCluster.Arn); err != nil {
			return err
		}
	}

	return nil
}
