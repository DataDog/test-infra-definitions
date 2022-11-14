package ec2

import (
	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/common/utils"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func NewDefaultEC2Instance(e aws.Environment, name, instanceType string) (*ec2.Instance, remote.ConnectionOutput, error) {
	awsInstance, err := NewEC2Instance(e, name, "", AMD64Arch, instanceType, e.DefaultKeyPairName(), "")
	if err != nil {
		return nil, remote.ConnectionOutput{}, err
	}

	connection := remote.ConnectionArgs{
		Host: awsInstance.PrivateIp,
	}
	if err := utils.ConfigureRemoteSSH("ubuntu", e.DefaultPrivateKeyPath(), e.DefaultPrivateKeyPassword(), "", &connection); err != nil {
		return nil, remote.ConnectionOutput{}, err
	}

	return awsInstance, connection.ToConnectionOutput(), nil
}

func NewEC2Instance(e aws.Environment, name, ami, arch, instanceType, keyPair, userData string) (*ec2.Instance, error) {
	var err error
	if ami == "" {
		ami, err = LatestUbuntuAMI(e, arch)
		if err != nil {
			return nil, err
		}
	}

	instance, err := ec2.NewInstance(e.Ctx, name, &ec2.InstanceArgs{
		Ami:                 pulumi.StringPtr(ami),
		SubnetId:            pulumi.StringPtr(e.DefaultSubnets()[0]),
		InstanceType:        pulumi.StringPtr(instanceType),
		VpcSecurityGroupIds: pulumi.ToStringArray(e.DefaultSecurityGroups()),
		KeyName:             pulumi.StringPtr(keyPair),
		UserData:            pulumi.StringPtr(userData),
	}, pulumi.Provider(e.Provider))
	return instance, err
}
