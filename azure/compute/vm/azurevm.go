package vm

import (
	"errors"

	"github.com/DataDog/test-infra-definitions/azure"
	"github.com/DataDog/test-infra-definitions/azure/compute"
	"github.com/DataDog/test-infra-definitions/command"
	"github.com/DataDog/test-infra-definitions/common/os"
	"github.com/DataDog/test-infra-definitions/common/vm"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type AzureVM struct {
	runner *command.Runner
}

// NewAzureVM creates a new azure instance. By default use WithOS(os.UbuntuOS, os.AMD64Arch).
func NewAzureVM(ctx *pulumi.Context, options ...func(*Params) error) (*AzureVM, error) {
	env, err := azure.AzureEnvironment(ctx)
	if err != nil {
		return nil, err
	}

	params, err := newParams(env, options...)
	if err != nil {
		return nil, err
	}

	if params.common.OS.GetOSType() != os.UbuntuOS {
		return nil, errors.New("only ubuntu is supported on Azure")
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

	runner, err := vm.InitVM(
		&env,
		publicIP.IpAddress.Elem(),
		params.common.OS,
		params.common.OptionalAgentInstallParams,
	)

	if err != nil {
		return nil, err
	}

	return &AzureVM{runner: runner}, nil
}
