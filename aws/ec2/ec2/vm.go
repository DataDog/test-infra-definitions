package ec2

import (
	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/command"
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type VM struct {
	context *pulumi.Context
	runner  *command.Runner

	CommonEnvironment *config.CommonEnvironment
	PackageManager    command.PackageManager
	FileManager       *command.FileManager
	DockerManager     *command.DockerManager
}

func NewVM(ctx *pulumi.Context) (*VM, error) {
	vm := &VM{
		context: ctx,
	}

	e, err := aws.AWSEnvironment(ctx)
	if err != nil {
		return nil, err
	}
	vm.CommonEnvironment = e.CommonEnvironment

	instance, conn, err := NewDefaultEC2Instance(e, "vm", e.DefaultInstanceType())
	if err != nil {
		return nil, err
	}

	vm.runner, err = command.NewRunner(*e.CommonEnvironment, e.CommonNamer.ResourceName("vm"), conn, func(r *command.Runner) (*remote.Command, error) {
		return command.WaitForCloudInit(ctx, r)
	})
	if err != nil {
		return nil, err
	}
	vm.PackageManager = command.NewAptManager(vm.runner)
	vm.DockerManager = command.NewDockerManager(vm.runner, vm.PackageManager)

	vm.FileManager = command.NewFileManager(vm.runner)

	e.Ctx.Export("instance-ip", instance.PrivateIp)
	e.Ctx.Export("connection", conn)

	return vm, nil
}

func NewDefaultEC2Instance(e aws.Environment, name, instanceType string) (*ec2.Instance, remote.ConnectionOutput, error) {
	awsInstance, err := NewEC2Instance(e, name, "", AMD64Arch, instanceType, e.DefaultKeyPairName(), "", "default")
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
		SubnetId:            pulumi.StringPtr(e.DefaultSubnets()[0]),
		InstanceType:        pulumi.StringPtr(instanceType),
		VpcSecurityGroupIds: pulumi.ToStringArray(e.DefaultSecurityGroups()),
		KeyName:             pulumi.StringPtr(keyPair),
		UserData:            pulumi.StringPtr(userData),
		Tenancy:             pulumi.StringPtr(tenancy),
		Tags: pulumi.StringMap{
			"Name": e.Namer.DisplayName(pulumi.String(name)),
		},
	}, pulumi.Provider(e.Provider))
	return instance, err
}
