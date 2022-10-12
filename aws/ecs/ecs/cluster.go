package ecs

import (
	"github.com/DataDog/test-infra-definitions/aws"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ecs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func CreateEcsCluster(e aws.Environment, name string) (*ecs.Cluster, error) {
	cluster, err := ecs.NewCluster(e.Ctx, name, &ecs.ClusterArgs{
		Name: pulumi.StringPtr(name),
		Configuration: &ecs.ClusterConfigurationArgs{
			ExecuteCommandConfiguration: &ecs.ClusterConfigurationExecuteCommandConfigurationArgs{
				KmsKeyId: pulumi.StringPtr(e.ECSExecKMSKeyID()),
			},
		},
	}, pulumi.Provider(e.Provider))
	if err != nil {
		return nil, err
	}

	return cluster, nil
}
