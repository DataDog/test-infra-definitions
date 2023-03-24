package fakeintake

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/aws"
	tidECS "github.com/DataDog/test-infra-definitions/aws/ecs"
	"github.com/cenkalti/backoff/v4"

	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/awsx"
	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/ecs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	containerName = "fakeintake"
	port          = 80
	oneSecond     = 1000
	sleepInterval = 1 * oneSecond
	maxRetries    = 120
)

func ECSFargateService(e aws.Environment, name string, clusterArn string) (ipAddress pulumi.StringOutput, err error) {
	taskDef, err := fargateLinuxTaskDefinition(e, name)
	if err != nil {
		return pulumi.StringOutput{}, err
	}
	fargateService, err := tidECS.FargateService(e, e.Namer.ResourceName(name), pulumi.String(clusterArn), taskDef.TaskDefinition.Arn())

	// Hack passing taskDef.TaskDefinition.Arn() to execute apply function
	// when taskDef has an ARN, thus it is defined on AWS side
	ipAddress = pulumi.All(taskDef.TaskDefinition.Arn(), fargateService.Service.Name()).ApplyT(func(args []any) (string, error) {
		var ipAddress string
		err := backoff.Retry(func() error {
			fmt.Println("waiting for task private ip")
			serviceName := args[1].(string)
			ecsClient, err := tidECS.NewECSClient(e.Ctx.Context(), e.Region())
			if err != nil {
				return err
			}
			ipAddress, err = ecsClient.GetTaskPrivateIP(clusterArn, serviceName)
			if err != nil {
				return err
			}
			fmt.Printf("task private ip found: %s\n", ipAddress)
			return err
		}, backoff.WithMaxRetries(backoff.NewConstantBackOff(sleepInterval), maxRetries))
		return ipAddress, err
	}).(pulumi.StringOutput)

	return ipAddress, err
}

func fargateLinuxTaskDefinition(e aws.Environment, name string) (*ecs.FargateTaskDefinition, error) {
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
		Cpu:         pulumi.IntPtr(100),
		Memory:      pulumi.IntPtr(512),
		Name:        pulumi.StringPtr(containerName),
		Image:       pulumi.StringPtr("public.ecr.aws/datadog/fakeintake:latest"),
		Essential:   pulumi.BoolPtr(true),
		MountPoints: ecs.TaskDefinitionMountPointArray{},
		Environment: ecs.TaskDefinitionKeyValuePairArray{},
		PortMappings: ecs.TaskDefinitionPortMappingArray{
			ecs.TaskDefinitionPortMappingArgs{
				ContainerPort: pulumi.Int(80),
				HostPort:      pulumi.Int(80),
				Protocol:      pulumi.StringPtr("tcp"),
			},
		},
		VolumesFrom: ecs.TaskDefinitionVolumeFromArray{},
	}
}
