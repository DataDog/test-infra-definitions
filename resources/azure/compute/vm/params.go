package vm

import (
	"github.com/DataDog/test-infra-definitions/common"
	commonos "github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/components/vm"
	"github.com/DataDog/test-infra-definitions/resources/azure"
	"github.com/DataDog/test-infra-definitions/resources/azure/compute/os"
)

type Params struct {
	env    azure.Environment
	common *vm.Params[commonos.OS]
}

func newParams(env azure.Environment, options ...func(*Params) error) (*Params, error) {
	commonParams, err := vm.NewParams[commonos.OS](env.CommonEnvironment)
	if err != nil {
		return nil, err
	}
	params := &Params{
		env:    env,
		common: commonParams,
	}
	if err := WithOS(os.UbuntuOS)(params); err != nil {
		return nil, err
	}
	return common.ApplyOption(params, options)
}

func (p *Params) GetCommonParams() *vm.Params[commonos.OS] {
	return p.common
}

func (p *Params) GetOS(osType os.Type) (commonos.OS, error) {
	return os.GetOS(p.env, osType)
}

// WithOS sets the OS. This function also set the instance type and the AMI.
func WithOS(osType os.Type) func(*Params) error {
	return vm.WithOS[commonos.OS, os.Type, *Params](osType)
}

// WithImageName set the name of the Image. `arch` and `osType` must match the AMI requirements.
func WithImageName(imageName string, arch commonos.Architecture, osType os.Type) func(*Params) error {
	return vm.WithImageName[commonos.OS, os.Type, *Params](imageName, arch, osType)
}

// WithArch set the architecture and the operating system.
func WithArch(osType os.Type, arch commonos.Architecture) func(*Params) error {
	return vm.WithArch[commonos.OS, os.Type, *Params](osType, arch)
}

// WithInstanceType set the instance type
func WithInstanceType(instanceType string) func(*Params) error {
	return vm.WithInstanceType[commonos.OS, os.Type, *Params](instanceType)
}

// WithUserData set the userdata for the instance. User data contains commands that are run at the startup of the instance.
func WithUserData(userData string) func(*Params) error {
	return vm.WithUserData[commonos.OS, os.Type, *Params](userData)
}

// WithName set the name of the instance
func WithName(name string) func(*Params) error {
	return vm.WithName[commonos.OS, os.Type, *Params](name)
}
