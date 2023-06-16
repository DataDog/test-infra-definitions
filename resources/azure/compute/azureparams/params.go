package azureparams

import (
	"github.com/DataDog/test-infra-definitions/common"
	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/components/vm"
	"github.com/DataDog/test-infra-definitions/resources/azure"
	"github.com/DataDog/test-infra-definitions/resources/azure/compute/azureos"
)

type Params struct {
	env    azure.Environment
	common *vm.Params[os.OS]
}

type Option = func(*Params) error

func NewParams(env azure.Environment, options ...Option) (*Params, error) {
	commonParams, err := vm.NewParams[os.OS](env.CommonEnvironment)
	if err != nil {
		return nil, err
	}
	params := &Params{
		env:    env,
		common: commonParams,
	}
	if err := WithOS(azureos.UbuntuOS)(params); err != nil {
		return nil, err
	}
	return common.ApplyOption(params, options)
}

func (p *Params) GetCommonParams() *vm.Params[os.OS] {
	return p.common
}

func (p *Params) getOS(osType azureos.Type) (os.OS, error) {
	return azureos.GetOS(p.env, osType)
}

// WithOS sets the OS. This function also set the instance type and the AMI.
func WithOS(osType azureos.Type) func(*Params) error {
	return func(p *Params) error {
		os, err := p.getOS(osType)
		if err != nil {
			return err
		}
		return p.common.SetOS(os)
	}
}

// WithImageName set the name of the Image. `arch` and `osType` must match the AMI requirements.
func WithImageName(imageName string, arch os.Architecture, osType azureos.Type) func(*Params) error {
	return func(p *Params) error {
		os, err := p.getOS(osType)
		if err != nil {
			return err
		}
		return p.common.SetImageName(imageName, arch, os)
	}
}

// WithArch set the architecture and the operating system.
func WithArch(osType azureos.Type, arch os.Architecture) func(*Params) error {
	return func(p *Params) error {
		os, err := p.getOS(osType)
		if err != nil {
			return err
		}
		return p.common.SetArch(os, arch)
	}
}

// WithInstanceType set the instance type
func WithInstanceType(instanceType string) func(*Params) error {
	return func(p *Params) error { return p.common.SetInstanceType(instanceType) }
}

// WithUserData set the userdata for the instance. User data contains commands that are run at the startup of the instance.
func WithUserData(userData string) func(*Params) error {
	return func(p *Params) error { return p.common.SetUserData(userData) }
}

// WithName set the name of the instance
func WithName(name string) func(*Params) error {
	return func(p *Params) error { return p.common.SetName(name) }
}
