package vm

import (
	"github.com/DataDog/test-infra-definitions/azure"
	"github.com/DataDog/test-infra-definitions/azure/compute"
	"github.com/DataDog/test-infra-definitions/common/vm"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// NewAzureVM creates a new azure instance. By default use WithOS(os.UbuntuOS, os.AMD64Arch).
func NewAzureVM(ctx *pulumi.Context, options ...func(*Params) error) (vm.VM, error) {
	env, err := azure.NewEnvironment(ctx)
	if err != nil {
		return nil, err
	}

	params, err := newParams(env, options...)
	if err != nil {
		return nil, err
	}

	_, publicIP, _, err := compute.NewLinuxInstance(
		env,
		env.CommonNamer.ResourceName(params.common.InstanceName),
		params.common.ImageName,
		params.common.InstanceType,
		pulumi.StringPtr(params.common.UserData),
	)

	if err != nil {
		return nil, err
	}

	return vm.NewVM(
		params.common.InstanceName,
		&env,
		publicIP.IpAddress.Elem(),
		params.common.OS,
	)
}
