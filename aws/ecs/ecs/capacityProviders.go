package ecs

import (
	"github.com/DataDog/test-infra-definitions/aws"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ecs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func NewCapacityProvider(e aws.Environment, name string, asgArn pulumi.StringInput) (*ecs.CapacityProvider, error) {
	return ecs.NewCapacityProvider(e.Ctx, name, &ecs.CapacityProviderArgs{
		AutoScalingGroupProvider: &ecs.CapacityProviderAutoScalingGroupProviderArgs{
			AutoScalingGroupArn: asgArn,
			ManagedScaling: &ecs.CapacityProviderAutoScalingGroupProviderManagedScalingArgs{
				Status: aws.DisabledString,
			},
			ManagedTerminationProtection: aws.DisabledString,
		},
	})
}

func NewClusterCapacityProvider(e aws.Environment, name string, clusterName pulumi.StringInput, capacityProviders pulumi.StringArray) (*ecs.ClusterCapacityProviders, error) {
	return ecs.NewClusterCapacityProviders(e.Ctx, name, &ecs.ClusterCapacityProvidersArgs{
		ClusterName:       clusterName,
		CapacityProviders: capacityProviders,
	})
}
