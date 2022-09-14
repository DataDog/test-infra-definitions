package ec2

import (
	"errors"

	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/common/config"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func CreateEC2Instance(ctx *pulumi.Context, env config.Environment, name, ami, arch, instanceType, keyPair, userData string) (*ec2.Instance, error) {
	var err error

	awsEnv, ok := env.(aws.Environment)
	if !ok {
		return nil, errors.New("creating EC2 instance is only supported on AWS Environments")
	}

	if ami == "" {
		ami, err = LatestUbuntuAMI(ctx, arch)
		if err != nil {
			return nil, err
		}
	}

	instance, err := ec2.NewInstance(ctx, name, &ec2.InstanceArgs{
		Ami:                 pulumi.StringPtr(ami),
		SubnetId:            pulumi.StringPtr(awsEnv.DefaultSubnet()),
		InstanceType:        pulumi.StringPtr(instanceType),
		VpcSecurityGroupIds: pulumi.ToStringArray(awsEnv.DefaultSecurityGroups()),
		KeyName:             pulumi.StringPtr(keyPair),
		UserData:            pulumi.StringPtr(userData),
	})
	return instance, err
}
