package ecs

import (
	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/aws/ec2/ec2"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ssm"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func NewECSOptimizedNodeGroup(ctx *pulumi.Context, environment aws.Environment) (pulumi.StringOutput, error) {
	ecsAmiParam, err := ssm.LookupParameter(ctx, &ssm.LookupParameterArgs{
		Name: "/aws/service/ecs/optimized-ami/amazon-linux-2/recommended",
	})
	if err != nil {
		return pulumi.StringOutput{}, err
	}

	lt, err := ec2.CreateLaunchTemplate(ctx, environment, ctx.Stack()+"-ecs-optimized-ng", ecsAmiParam.Value, environment.DefaultInstanceType(), environment.DefaultKeyPairName(), "")
	if err != nil {
		return pulumi.StringOutput{}, err
	}

	asg, err := ec2.NewAutoscalingGroup(ctx, environment, ctx.Stack()+"-ecs-optimized-ng", lt.Arn, 1, 1, 1)
	if err != nil {
		return pulumi.StringOutput{}, err
	}

	cp, err := NewCapacityProvider(ctx, environment, ctx.Stack()+"-ecs-optimized-ng", asg.Arn)
	if err != nil {
		return pulumi.StringOutput{}, err
	}

	return cp.Name, nil
}
