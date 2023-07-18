package ec2

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/resources/aws"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func NewEC2Instance(e aws.Environment, name, ami, arch, instanceType, keyPair, userData, tenancy string) (*ec2.Instance, error) {
	var err error
	if ami == "" {
		ami, err = LatestUbuntuAMI(e, arch)
		if err != nil {
			return nil, err
		}
	}

	instance, err := ec2.NewInstance(e.Ctx, e.Namer.ResourceName(name), &ec2.InstanceArgs{
		Ami:                 pulumi.StringPtr(ami),
		SubnetId:            e.RandomSubnets().Index(pulumi.Int(0)),
		InstanceType:        pulumi.StringPtr(instanceType),
		VpcSecurityGroupIds: pulumi.ToStringArray(e.DefaultSecurityGroups()),
		KeyName:             pulumi.StringPtr(keyPair),
		UserData:            pulumi.StringPtr(userData),
		Tenancy:             pulumi.StringPtr(tenancy),
		RootBlockDevice: ec2.InstanceRootBlockDeviceArgs{
			VolumeSize: pulumi.Int(e.DefaultInstanceStorageSize()),
		},
		Tags: pulumi.StringMap{
			"Name": e.Namer.DisplayName(255, pulumi.String(name)),
		},
		InstanceInitiatedShutdownBehavior: pulumi.String(e.DefaultShutdownBehavior()),
	}, e.WithProviders(config.ProviderAWS))
	return instance, err
}
