package ec2

import (
	"strconv"

	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/autoscaling"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func NewAutoscalingGroup(e aws.Environment, name string,
	launchTemplateID pulumi.StringInput,
	launchTemplateVersion pulumi.IntInput,
	desired, min, max int,
) (*autoscaling.Group, error) {
	return autoscaling.NewGroup(e.Ctx, e.Namer.ResourceName(name), &autoscaling.GroupArgs{
		NamePrefix:      e.CommonNamer.DisplayName(pulumi.String(name)),
		DesiredCapacity: pulumi.Int(desired),
		MinSize:         pulumi.Int(min),
		MaxSize:         pulumi.Int(max),
		LaunchTemplate: autoscaling.GroupLaunchTemplateArgs{
			Id:      launchTemplateID,
			Version: launchTemplateVersion.ToIntOutput().ApplyT(func(v int) pulumi.String { return pulumi.String(strconv.Itoa(v)) }).(pulumi.StringInput),
		},
		CapacityRebalance: pulumi.Bool(true),
		InstanceRefresh: autoscaling.GroupInstanceRefreshArgs{
			Strategy: pulumi.String("Rolling"),
			Preferences: autoscaling.GroupInstanceRefreshPreferencesArgs{
				MinHealthyPercentage: pulumi.Int(0),
			},
		},
	}, pulumi.Provider(e.Provider))
}
