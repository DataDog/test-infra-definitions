package ecs

import (
	"github.com/DataDog/test-infra-definitions/aws"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ecs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func CreateEcsCluster(e aws.Environment, name string) (*ecs.Cluster, error) {
	return ecs.NewCluster(e.Ctx, e.Namer.ResourceName(name), &ecs.ClusterArgs{
		Name: e.Namer.DisplayName(pulumi.String(name)),
		Configuration: &ecs.ClusterConfigurationArgs{
			ExecuteCommandConfiguration: &ecs.ClusterConfigurationExecuteCommandConfigurationArgs{
				KmsKeyId: pulumi.StringPtr(e.ECSExecKMSKeyID()),
			},
		},
	}, pulumi.Provider(e.Provider))
}
