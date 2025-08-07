package ec2

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/resources/aws"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type LaunchTemplateArgs struct {
	InstanceType      pulumi.StringInput
	ImageID           pulumi.StringInput
	UserData          pulumi.StringInput
	AssociatePublicIp bool
	KeyName           pulumi.StringInput
	SecurityGroupIDs  pulumi.StringArrayInput
}

func NewEC2LaunchTemplate(e aws.Environment, name string, args *LaunchTemplateArgs) (*ec2.LaunchTemplate, error) {
	launchTemplate, err := ec2.NewLaunchTemplate(e.Ctx(), e.Namer.ResourceName(name), &ec2.LaunchTemplateArgs{
		NamePrefix:   e.CommonNamer().DisplayName(128, pulumi.String(name)),
		InstanceType: args.InstanceType,
		ImageId:      args.ImageID,
		UserData:     args.UserData,
		KeyName:      args.KeyName,
		NetworkInterfaces: ec2.LaunchTemplateNetworkInterfaceArray{
			ec2.LaunchTemplateNetworkInterfaceArgs{
				AssociatePublicIpAddress: pulumi.StringPtr(fmt.Sprintf("%v", args.AssociatePublicIp)),
				DeleteOnTermination:      pulumi.StringPtr("true"),
				SecurityGroups:           args.SecurityGroupIDs,
			},
		},
		UpdateDefaultVersion: pulumi.BoolPtr(true),
	}, e.WithProviders(config.ProviderAWS))

	return launchTemplate, err
}
