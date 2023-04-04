package fakeintake

import (
	"github.com/DataDog/test-infra-definitions/aws"

	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/awsx"
	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/ecs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	containerName = "fakeintake"
	port          = 80
)

const (
	FargateClusterArn     = "arn:aws:ecs:us-east-1:601427279990:cluster/fakeintake-ecs"
	DefaultFakeintakePort = 80
)

func FargateLinuxTaskDefinition(e aws.Environment, name string) (*ecs.FargateTaskDefinition, error) {
	return ecs.NewFargateTaskDefinition(e.Ctx, e.Namer.ResourceName(name), &ecs.FargateTaskDefinitionArgs{
		Containers: map[string]ecs.TaskDefinitionContainerDefinitionArgs{
			containerName: *fargateLinuxContainerDefinition(),
		},
		Cpu:    pulumi.StringPtr("256"),
		Memory: pulumi.StringPtr("512"),
		ExecutionRole: &awsx.DefaultRoleWithPolicyArgs{
			RoleArn: pulumi.StringPtr(e.ECSTaskExecutionRole()),
		},
		TaskRole: &awsx.DefaultRoleWithPolicyArgs{
			RoleArn: pulumi.StringPtr(e.ECSTaskRole()),
		},
		Family: e.CommonNamer.DisplayName(pulumi.String("fakeintake-ecs")),
	}, e.ResourceProvidersOption())
}

func fargateLinuxContainerDefinition() *ecs.TaskDefinitionContainerDefinitionArgs {
	return &ecs.TaskDefinitionContainerDefinitionArgs{
		Name:        pulumi.StringPtr(containerName),
		Image:       pulumi.StringPtr("public.ecr.aws/datadog/fakeintake:latest"),
		Essential:   pulumi.BoolPtr(true),
		MountPoints: ecs.TaskDefinitionMountPointArray{},
		Environment: ecs.TaskDefinitionKeyValuePairArray{},
		PortMappings: ecs.TaskDefinitionPortMappingArray{
			ecs.TaskDefinitionPortMappingArgs{
				ContainerPort: pulumi.Int(port),
				HostPort:      pulumi.Int(port),
				Protocol:      pulumi.StringPtr("tcp"),
			},
		},
		VolumesFrom: ecs.TaskDefinitionVolumeFromArray{},
	}
}
