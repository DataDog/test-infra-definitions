package vm

import (
	commonvm "github.com/DataDog/test-infra-definitions/components/vm"
	"github.com/DataDog/test-infra-definitions/resources/azure"
	"github.com/DataDog/test-infra-definitions/resources/azure/compute"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// NewAzureVM creates a new azure instance. By default use WithOS(os.UbuntuOS, os.AMD64Arch).
func NewAzureVM(ctx *pulumi.Context, options ...func(*Params) error) (commonvm.VM, error) {
	return newVM(ctx, options...)
}

// NewUnixAzureVM creates a new azure instance. By default use WithOS(os.UbuntuOS, os.AMD64Arch).
// The returned vm provides additional methods compared to NewAzureVM
func NewUnixAzureVM(ctx *pulumi.Context, options ...func(*Params) error) (*commonvm.UnixVM, error) {
	vm, err := newVM(ctx, options...)
	if err != nil {
		return nil, err
	}
	return commonvm.NewUnixVM(vm)
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

	vm, publicIP, _, err := compute.NewLinuxInstance(
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
		vm,
		&env,
		publicIP.IpAddress.Elem(),
		params.common.OS,
	)
}
