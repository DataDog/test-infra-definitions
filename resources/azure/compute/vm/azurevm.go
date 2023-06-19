package vm

import (
	commonvm "github.com/DataDog/test-infra-definitions/components/vm"
	"github.com/DataDog/test-infra-definitions/resources/azure"
	"github.com/DataDog/test-infra-definitions/resources/azure/compute"
	"github.com/DataDog/test-infra-definitions/resources/azure/compute/azureparams"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// NewAzureVM creates a new azure instance. By default use WithOS(os.UbuntuOS, os.AMD64Arch).
func NewAzureVM(ctx *pulumi.Context, options ...azureparams.Option) (commonvm.VM, error) {
	return newVM(ctx, options...)
}

// NewUnixAzureVM creates a new azure instance. By default use WithOS(os.UbuntuOS, os.AMD64Arch).
// The returned vm provides additional methods compared to NewAzureVM
func NewUnixAzureVM(ctx *pulumi.Context, options ...azureparams.Option) (*commonvm.UnixVM, error) {
	vm, err := newVM(ctx, options...)
	if err != nil {
		return nil, err
	}
	return commonvm.NewUnixVM(vm)
}

func newVM(ctx *pulumi.Context, options ...azureparams.Option) (commonvm.VM, error) {
	env, err := azure.NewEnvironment(ctx)
	if err != nil {
		return nil, err
	}

	params, err := azureparams.NewParams(env, options...)
	if err != nil {
		return nil, err
	}
	commonParams := params.GetCommonParams()
	vm, publicIP, _, err := compute.NewLinuxInstance(
		env,
		env.CommonNamer.ResourceName(commonParams.InstanceName),
		commonParams.ImageName,
		commonParams.InstanceType,
		pulumi.StringPtr(commonParams.UserData),
	)
	if err != nil {
		return nil, err
	}

	return commonvm.NewGenericVM(
		commonParams.InstanceName,
		vm,
		&env,
		publicIP.IpAddress.Elem(),
		commonParams.OS,
	)
}
