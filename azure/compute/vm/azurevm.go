package vm

import (
	"github.com/DataDog/test-infra-definitions/azure"
	"github.com/DataDog/test-infra-definitions/azure/compute"
	commonvm "github.com/DataDog/test-infra-definitions/common/vm"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// NewAzureVM creates a new azure instance. By default use WithOS(os.UbuntuOS, os.AMD64Arch).
func NewAzureVM(ctx *pulumi.Context, options ...func(*Params) error) (commonvm.VM, error) {
	return newVM(ctx, options...)
}

// NewNixAzureVM creates a new azure instance. By default use WithOS(os.UbuntuOS, os.AMD64Arch).
// The returned vm provides additional methods compared to NewAzureVM
func NewNixAzureVM(ctx *pulumi.Context, options ...func(*Params) error) (*commonvm.NixVM, error) {
	vm, err := newVM(ctx, options...)
	if err != nil {
		return nil, err
	}
	return commonvm.NewNixVM(vm)
}

func newVM(ctx *pulumi.Context, options ...func(*Params) error) (commonvm.VM, error) {
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

	return commonvm.NewGenericVM(
		params.common.InstanceName,
		&env,
		publicIP.IpAddress.Elem(),
		params.common.OS,
	)
}
