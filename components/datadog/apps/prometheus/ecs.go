package prometheus

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/resources/aws"

	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/awsx"
	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/ecs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type EcsComponent struct {
	pulumi.ResourceState
}

func EcsAppDefinition(e aws.Environment, clusterArn pulumi.StringInput, opts ...pulumi.ResourceOption) (*EcsComponent, error) {
	namer := e.Namer.WithPrefix("prometheus")
	opts = append(opts, e.WithProviders(config.ProviderAWS, config.ProviderAWSX))

	ecsComponent := &EcsComponent{}
	if err := e.Ctx.RegisterComponentResource("dd:apps", namer.ResourceName("grp"), ecsComponent, opts...); err != nil {
		return nil, err
	}

	opts = append(opts, pulumi.Parent(ecsComponent))

	if _, err := ecs.NewEC2Service(e.Ctx, namer.ResourceName("server"), &ecs.EC2ServiceArgs{
		Name:                 e.CommonNamer.DisplayName(pulumi.String("prometheus")),
		Cluster:              clusterArn,
		DesiredCount:         pulumi.IntPtr(1),
		EnableExecuteCommand: pulumi.BoolPtr(true),
		TaskDefinitionArgs: &ecs.EC2ServiceTaskDefinitionArgs{
			Containers: map[string]ecs.TaskDefinitionContainerDefinitionArgs{
				"prometheus": {
					Name:  pulumi.StringPtr("prometheus"),
					Image: pulumi.StringPtr("ghcr.io/datadog/apps-prometheus:main"),
					DockerLabels: pulumi.StringMap{
						"com.datadoghq.ad.checks": pulumi.String(jsonMustMarshal(
							map[string]interface{}{
								"openmetrics": map[string]interface{}{
									"init_config": map[string]interface{}{},
									"instances": []map[string]interface{}{
										{
											"openmetrics_endpoint": "http://%%host%%:8080/metrics",
											"namespace":            "prometheus",
											"metrics": []string{
												"prom_.*",
											},
										},
									},
								},
							},
						)),
					},
					Cpu:    pulumi.IntPtr(50),
					Memory: pulumi.IntPtr(32),
				},
			},
			ExecutionRole: &awsx.DefaultRoleWithPolicyArgs{
				RoleArn: pulumi.StringPtr(e.ECSTaskExecutionRole()),
			},
			TaskRole: &awsx.DefaultRoleWithPolicyArgs{
				RoleArn: pulumi.StringPtr(e.ECSTaskRole()),
			},
			NetworkMode: pulumi.StringPtr("bridge"),
			Family:      e.CommonNamer.DisplayName(pulumi.String("prometheus-ec2")),
		},
	}, opts...); err != nil {
		return nil, err
	}

	return ecsComponent, nil
}
