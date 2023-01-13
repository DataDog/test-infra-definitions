package ec2vm

import (
	"github.com/DataDog/test-infra-definitions/aws"
	awsEc2 "github.com/DataDog/test-infra-definitions/aws/ec2/ec2"
	"github.com/DataDog/test-infra-definitions/command"
	"github.com/DataDog/test-infra-definitions/common/vm"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Ec2VM struct {
	runner *command.Runner
}

// NewEc2VM creates a new EC2 instance. By default use WithOS(os.UbuntuOS, os.AMD64Arch).
func NewEc2VM(ctx *pulumi.Context, options ...func(*Params) error) (*Ec2VM, error) {
	env, err := aws.AWSEnvironment(ctx)
	if err != nil {
		return nil, err
	}

	params, err := newParams(env, options...)
	if err != nil {
		return nil, err
	}

	os := params.common.OS
	instance, err := awsEc2.NewEC2Instance(
		env,
		env.CommonNamer.ResourceName(params.common.GetInstanceNameOrDefault("ec2-instance")),
		params.common.ImageName,
		os.GetAMIArch(params.common.Arch),
		params.common.InstanceType,
		params.keyPair,
		params.common.UserData,
		os.GetTenancy())

	if err != nil {
		return nil, err
	}

	runner, err := vm.InitVM(
		&env,
		instance.PrivateIp,
		os,
		params.common.OptionalAgentInstallParams,
	)

	if err != nil {
		return nil, err
	}

	return &Ec2VM{runner: runner}, nil
}
