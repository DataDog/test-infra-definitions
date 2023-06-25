package ec2params

import (
	"github.com/DataDog/test-infra-definitions/common"
	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/components/vm"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/vm/ec2os"
)

// Params defines the parameters for a virtual machine.
// The Params configuration uses the [Functional options pattern].
//
// The available options are:
//   - [WithOS]
//   - [WithImageName]
//   - [WithArch]
//   - [WithInstanceType]
//   - [WithUserData]
//   - [WithName]
//
// [Functional options pattern]: https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis
type Params struct {
	env    aws.Environment
	common *vm.Params[ec2os.OS]
}

type Option = func(*Params) error

func NewParams(env aws.Environment, options ...Option) (*Params, error) {
	commonParams, err := vm.NewParams[ec2os.OS]()
	if err != nil {
		return nil, err
	}
	params := &Params{
		env:    env,
		common: commonParams,
	}

	// Can be overrided later if the caller uses WithOS.
	if err := WithOS(ec2os.UbuntuOS)(params); err != nil {
		return nil, err
	}
	return common.ApplyOption(params, options)
}

func (p *Params) GetCommonParams() *vm.Params[ec2os.OS] {
	return p.common
}

func (p *Params) getOS(osType ec2os.Type) (ec2os.OS, error) {
	return ec2os.GetOS(p.env, osType)
}

// WithOS sets the OS. This function also set the instance type and the AMI.
func WithOS(osType ec2os.Type) func(*Params) error {
	return func(p *Params) error {
		os, err := p.getOS(osType)
		if err != nil {
			return err
		}
		return p.common.SetOS(os)
	}
}

// WithImageName set the name of the Image. `arch` and `osType` must match the AMI requirements.
func WithImageName(imageName string, arch os.Architecture, osType ec2os.Type) func(*Params) error {
	return func(p *Params) error {
		os, err := p.getOS(osType)
		if err != nil {
			return err
		}
		return p.common.SetImageName(imageName, arch, os)
	}
}

// WithArch set the architecture and the operating system.
func WithArch(osType ec2os.Type, arch os.Architecture) func(*Params) error {
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
