package ecs

import (
	"github.com/DataDog/test-infra-definitions/aws"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ecs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func NewCapacityProvider(ctx *pulumi.Context, environment aws.Environment, name string, asgArn pulumi.StringInput) (*ecs.CapacityProvider, error) {
	return ecs.NewCapacityProvider(ctx, name, &ecs.CapacityProviderArgs{
		AutoScalingGroupProvider: &ecs.CapacityProviderAutoScalingGroupProviderArgs{
			AutoScalingGroupArn: asgArn,
			ManagedScaling: &ecs.CapacityProviderAutoScalingGroupProviderManagedScalingArgs{
				Status: aws.DisabledString,
			},
			ManagedTerminationProtection: aws.EnabledString,
		},
	})
}
