package dogstatsd

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/resources/aws"

	classicECS "github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ecs"
	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/awsx"
	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/ecs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type EcsComponent struct {
	pulumi.ResourceState
}

func EcsAppDefinition(e aws.Environment, clusterArn pulumi.StringInput, opts ...pulumi.ResourceOption) (*EcsComponent, error) {
	namer := e.Namer.WithPrefix("dogstatsd")
	opts = append(opts, e.WithProviders(config.ProviderAWS, config.ProviderAWSX))

	ecsComponent := &EcsComponent{}
	if err := e.Ctx.RegisterComponentResource("dd:apps", namer.ResourceName("grp"), ecsComponent, opts...); err != nil {
		return nil, err
	}

	opts = append(opts, pulumi.Parent(ecsComponent))

	if _, err := ecs.NewEC2Service(e.Ctx, namer.ResourceName("uds"), &ecs.EC2ServiceArgs{
		Name:                 e.CommonNamer.DisplayName(pulumi.String("dogstatsd-uds")),
		Cluster:              clusterArn,
		DesiredCount:         pulumi.IntPtr(1),
		EnableExecuteCommand: pulumi.BoolPtr(true),
		TaskDefinitionArgs: &ecs.EC2ServiceTaskDefinitionArgs{
			Containers: map[string]ecs.TaskDefinitionContainerDefinitionArgs{
				"dogstatsd": {
					Name:  pulumi.StringPtr("dogstatsd"),
					Image: pulumi.StringPtr("ghcr.io/datadog/apps-dogstatsd:main"),
					Environment: ecs.TaskDefinitionKeyValuePairArray{
						ecs.TaskDefinitionKeyValuePairArgs{
							Name:  pulumi.StringPtr("STATSD_URL"),
							Value: pulumi.StringPtr("unix:///var/run/datadog/dsd.socket"),
						},
					},
					Cpu:    pulumi.IntPtr(50),
					Memory: pulumi.IntPtr(32),
					MountPoints: ecs.TaskDefinitionMountPointArray{
						ecs.TaskDefinitionMountPointArgs{
							SourceVolume:  pulumi.StringPtr("dd-sockets"),
							ContainerPath: pulumi.StringPtr("/var/run/datadog"),
							ReadOnly:      pulumi.BoolPtr(true),
						},
					},
				},
			},
			ExecutionRole: &awsx.DefaultRoleWithPolicyArgs{
				RoleArn: pulumi.StringPtr(e.ECSTaskExecutionRole()),
			},
			TaskRole: &awsx.DefaultRoleWithPolicyArgs{
				RoleArn: pulumi.StringPtr(e.ECSTaskRole()),
			},
			NetworkMode: pulumi.StringPtr("none"),
			Family:      e.CommonNamer.DisplayName(pulumi.String("dogstatsd-uds-ec2")),
			Volumes: classicECS.TaskDefinitionVolumeArray{
				classicECS.TaskDefinitionVolumeArgs{
					Name:     pulumi.String("dd-sockets"),
					HostPath: pulumi.StringPtr("/var/run/datadog"),
				},
			},
		},
	}, opts...); err != nil {
		return nil, err
	}

	return ecsComponent, nil
}
