package ecs

import (
	awscommon "github.com/vboulineau/pulumi-definitions/aws-common"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ecs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func Run(ctx *pulumi.Context) error {
	resourcesPrefix := config.Require(ctx, "PREFIX")
	environment, err := awscommon.GetEnvironmentFromConfig(ctx)
	if err != nil {
		return err
	}

	ecsCluster, err := createEcsCluster(ctx, environment, resourcesPrefix)
	if err != nil {
		return err
	}
	cps := pulumi.StringArray{pulumi.String("FARGATE")}

	ecs.NewClusterCapacityProviders(ctx, getResourceName(resourcesPrefix, "cluster-cp"), &ecs.ClusterCapacityProvidersArgs{
		ClusterName:       ecsCluster.Name,
		CapacityProviders: cps,
	})

	if err != nil {
		return err
	}

	return nil
}

func createEcsCluster(ctx *pulumi.Context, environment awscommon.Environment, prefix string) (*ecs.Cluster, error) {
	return ecs.NewCluster(ctx, getResourceName(prefix, "cluster"), &ecs.ClusterArgs{
		Configuration: &ecs.ClusterConfigurationArgs{
			ExecuteCommandConfiguration: &ecs.ClusterConfigurationExecuteCommandConfigurationArgs{
				KmsKeyId: pulumi.StringPtr(environment.ECSExecKMSKeyID()),
			},
		},
	})
}

func getResourceName(prefix, resourceName string) string {
	return prefix + "-" + resourceName
}
