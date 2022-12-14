package ec2

import (
	"github.com/DataDog/test-infra-definitions/aws"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func CreateLaunchTemplate(e aws.Environment, name string, ami, instanceType, iamProfileArn, keyPair, userData pulumi.StringInput) (*ec2.LaunchTemplate, error) {
	launchTemplate, err := ec2.NewLaunchTemplate(e.Ctx, e.Namer.ResourceName(name), &ec2.LaunchTemplateArgs{
		ImageId:      ami,
		NamePrefix:   e.CommonNamer.DisplayName(pulumi.String(name)),
		InstanceType: instanceType,
		IamInstanceProfile: ec2.LaunchTemplateIamInstanceProfileArgs{
			Arn: iamProfileArn,
		},
		NetworkInterfaces: ec2.LaunchTemplateNetworkInterfaceArray{
			ec2.LaunchTemplateNetworkInterfaceArgs{
				SubnetId:                 pulumi.StringPtr(e.DefaultSubnets()[0]),
				DeleteOnTermination:      pulumi.StringPtr("true"),
				AssociatePublicIpAddress: pulumi.StringPtr("false"),
				SecurityGroups:           pulumi.ToStringArray(e.DefaultSecurityGroups()),
			},
		},
		BlockDeviceMappings: ec2.LaunchTemplateBlockDeviceMappingArray{
			ec2.LaunchTemplateBlockDeviceMappingArgs{},
		},
		KeyName:              keyPair,
		UserData:             userData,
		UpdateDefaultVersion: pulumi.BoolPtr(true),
	}, pulumi.Provider(e.Provider))
	return launchTemplate, err
}
