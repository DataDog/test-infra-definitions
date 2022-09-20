package ec2

import (
	"github.com/DataDog/test-infra-definitions/aws"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func CreateEC2Instance(ctx *pulumi.Context, name, ami, arch, instanceType, keyPair, userData string) (*ec2.Instance, error) {
	var err error
	awsEnv := aws.AWSEnvironment(ctx)

	if ami == "" {
		ami, err = LatestUbuntuAMI(ctx, arch)
		if err != nil {
			return nil, err
		}
	}

	instance, err := ec2.NewInstance(ctx, name, &ec2.InstanceArgs{
		Ami:                 pulumi.StringPtr(ami),
		SubnetId:            pulumi.StringPtr(awsEnv.DefaultSubnets()[0]),
		InstanceType:        pulumi.StringPtr(instanceType),
		VpcSecurityGroupIds: pulumi.ToStringArray(awsEnv.DefaultSecurityGroups()),
		KeyName:             pulumi.StringPtr(keyPair),
		UserData:            pulumi.StringPtr(userData),
	})
	return instance, err
}
