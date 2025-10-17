package busybox

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	ecsComp "github.com/DataDog/test-infra-definitions/components/ecs"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/pulumi/pulumi-awsx/sdk/v2/go/awsx/awsx"
	"github.com/pulumi/pulumi-awsx/sdk/v2/go/awsx/ecs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func EcsAppDefinition(e aws.Environment, clusterArn pulumi.StringInput, opts ...pulumi.ResourceOption) (*ecsComp.Workload, error) {
	namer := e.Namer.WithPrefix("busybox").WithPrefix("ec2")
	opts = append(opts, e.WithProviders(config.ProviderAWS, config.ProviderAWSX))

	ecsComponent := &ecsComp.Workload{}
	if err := e.Ctx().RegisterComponentResource("dd:apps", namer.ResourceName("noop"), ecsComponent, opts...); err != nil {
		return nil, err
	}

	opts = append(opts, pulumi.Parent(ecsComponent))

	if _, err := ecs.NewEC2Service(e.Ctx(), namer.ResourceName("noop"), &ecs.EC2ServiceArgs{
		Name:                 e.CommonNamer().DisplayName(255, pulumi.String("busybox"), pulumi.String("ec2")),
		Cluster:              clusterArn,
		DesiredCount:         pulumi.IntPtr(1),
		EnableExecuteCommand: pulumi.BoolPtr(true),
		TaskDefinitionArgs: &ecs.EC2ServiceTaskDefinitionArgs{
			Containers: map[string]ecs.TaskDefinitionContainerDefinitionArgs{
				"busybox": {
					Name:  pulumi.String("busybox"),
					Image: pulumi.String("busybox:latest"),
					Command: pulumi.StringArray{
						pulumi.String("sh"),
						pulumi.String("-c"),
						pulumi.String("while true; do sleep 3600; done"),
					},
					Cpu:    pulumi.IntPtr(50),
					Memory: pulumi.IntPtr(32),
					DockerLabels: pulumi.StringMap{
						"com.datadoghq.ad.checks": pulumi.String(utils.JSONMustMarshal(
							map[string]interface{}{
								"busybox_unscheduled_check": map[string]interface{}{
									"init_config": map[string]interface{}{},
									"instances": []map[string]interface{}{
										{
											"name": "busybox_unscheduled_check",
										},
									},
								},
							},
						)),
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
			Family:      e.CommonNamer().DisplayName(255, pulumi.ToStringArray([]string{"busybox", "ec2"})...),
		},
	}, opts...); err != nil {
		return nil, err
	}

	return ecsComponent, nil
}
