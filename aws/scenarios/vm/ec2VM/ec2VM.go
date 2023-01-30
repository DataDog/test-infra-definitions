package ec2vm

import (
	"github.com/DataDog/test-infra-definitions/aws"
	awsEc2 "github.com/DataDog/test-infra-definitions/aws/ec2/ec2"
	"github.com/DataDog/test-infra-definitions/common/vm"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// NewEc2VM creates a new EC2 instance. By default use WithOS(os.UbuntuOS, os.AMD64Arch).
func NewEc2VM(ctx *pulumi.Context, options ...func(*Params) error) (vm.VM, error) {
	env, err := aws.NewEnvironment(ctx)
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
		env.CommonNamer.ResourceName(params.common.ImageName),
		params.common.ImageName,
		os.GetAMIArch(params.common.Arch),
		params.common.InstanceType,
		params.keyPair,
		params.common.UserData,
		os.GetTenancy())

	if err != nil {
		return nil, err
	}

	return vm.NewVM(
		params.common.InstanceName,
		&env,
		instance.PrivateIp,
		os,
		params.common.OptionalAgentInstallParams,
	)
}
