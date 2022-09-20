package ec2

import (
	"github.com/DataDog/test-infra-definitions/aws"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func CreateLaunchTemplate(ctx *pulumi.Context, environment aws.Environment, name, ami, instanceType, keyPair, userData string) (*ec2.LaunchTemplate, error) {
	launchTemplate, err := ec2.NewLaunchTemplate(ctx, name, &ec2.LaunchTemplateArgs{
		ImageId:      pulumi.StringPtr(ami),
		Name:         pulumi.StringPtr(name),
		InstanceType: pulumi.StringPtr(instanceType),
		NetworkInterfaces: ec2.LaunchTemplateNetworkInterfaceArray{
			ec2.LaunchTemplateNetworkInterfaceArgs{
				SubnetId:                 pulumi.StringPtr(environment.DefaultSubnets()[0]),
				DeleteOnTermination:      pulumi.StringPtr("true"),
				AssociatePublicIpAddress: pulumi.StringPtr("false"),
				SecurityGroups:           pulumi.ToStringArray(environment.DefaultSecurityGroups()),
			},
		},
		KeyName:  pulumi.StringPtr(keyPair),
		UserData: pulumi.StringPtr(userData),
	})
	return launchTemplate, err
}
