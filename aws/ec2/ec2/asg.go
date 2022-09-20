package ec2

import (
	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/autoscaling"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func NewAutoscalingGroup(ctx *pulumi.Context, environment aws.Environment, name string,
	launchTemplateARN pulumi.StringInput,
	desired, min, max int,
) (*autoscaling.Group, error) {
	return autoscaling.NewGroup(ctx, name, &autoscaling.GroupArgs{
		Name:            pulumi.String(name),
		DesiredCapacity: pulumi.Int(desired),
		MinSize:         pulumi.Int(min),
		MaxSize:         pulumi.Int(max),
		LaunchTemplate: autoscaling.GroupLaunchTemplateArgs{
			Id:      launchTemplateARN,
			Version: pulumi.String("$Latest"),
		},
		CapacityRebalance: pulumi.Bool(true),
	})
}
