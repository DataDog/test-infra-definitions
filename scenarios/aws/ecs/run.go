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

	var apiKeyParam *ssm.Parameter
	var fakeIntake *fakeintakeComp.Fakeintake
	// Create task and service
	if awsEnv.AgentDeploy() {
		if awsEnv.GetCommonEnvironment().AgentUseFakeintake() {
			fakeIntakeOptions := []fakeintake.Option{}
			if awsEnv.InfraShouldDeployFakeintakeWithLB() {
				fakeIntakeOptions = append(fakeIntakeOptions, fakeintake.WithLoadBalancer())
			}

			if fakeIntake, err = fakeintake.NewECSFargateInstance(awsEnv, "ecs", fakeIntakeOptions...); err != nil {
				return err
			}
			if err := fakeIntake.Export(awsEnv.Ctx, nil); err != nil {
				return err
			}
		}
		apiKeyParam, err = ssm.NewParameter(ctx, awsEnv.Namer.ResourceName("agent-apikey"), &ssm.ParameterArgs{
			Name:      awsEnv.CommonNamer.DisplayName(1011, pulumi.String("agent-apikey")),
			Type:      ssm.ParameterTypeSecureString,
			Overwrite: pulumi.Bool(true),
			Value:     awsEnv.AgentAPIKey(),
		}, awsEnv.WithProviders(config.ProviderAWS))
		if err != nil {
			return err
		}

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

	// Deploy Fargate Agents
	if awsEnv.TestingWorkloadDeploy() && awsEnv.AgentDeploy() {
		if _, err := redis.FargateAppDefinition(awsEnv, ecsCluster.Arn, apiKeyParam.Name, fakeIntake); err != nil {
			return err
		}

		if _, err = nginx.FargateAppDefinition(awsEnv, ecsCluster.Arn, apiKeyParam.Name, fakeIntake); err != nil {
			return err
		}
	}

	return nil
}
