package ecs

import (
	"github.com/DataDog/test-infra-definitions/aws"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ecs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func CreateEcsCluster(ctx *pulumi.Context, environment aws.Environment, capacityProviders pulumi.StringArray) (*ecs.Cluster, error) {
	cluster, err := ecs.NewCluster(ctx, ctx.Stack(), &ecs.ClusterArgs{
		Configuration: &ecs.ClusterConfigurationArgs{
			ExecuteCommandConfiguration: &ecs.ClusterConfigurationExecuteCommandConfigurationArgs{
				KmsKeyId: pulumi.StringPtr(environment.ECSExecKMSKeyID()),
			},
		},
	})
	if err != nil {
		return nil, err
	}

	ecs.NewClusterCapacityProviders(ctx, ctx.Stack()+"-cp", &ecs.ClusterCapacityProvidersArgs{
		ClusterName:       cluster.Name,
		CapacityProviders: capacityProviders,
	})

	return cluster, nil
}
