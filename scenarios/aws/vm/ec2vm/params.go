package ec2vm

import (
	"fmt"
	"strings"

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

func newParams(env aws.Environment, options ...func(*Params) error) (*Params, error) {
	commonParams, err := vm.NewParams[ec2os.OS](env.CommonEnvironment)
	if err != nil {
		return nil, err
	}
	params := &Params{
		env:    env,
		common: commonParams,
	}

	// Can be overrided later if the caller uses WithOS.
	if err := params.useDefaultOS(); err != nil {
		return nil, err
	}
	return common.ApplyOption(params, options)
}

func (p *Params) getOS(osType ec2os.Type) (ec2os.OS, error) {
	return ec2os.GetOS(p.env, osType)
}

func (p *Params) useDefaultOS() error {
	var osType ec2os.Type

	osTypeStr := strings.ToLower(p.env.InfraOSFamily())
	switch osTypeStr {
	case "windows":
		osType = ec2os.WindowsOS
	case "ubuntu":
		osType = ec2os.UbuntuOS
	case "amazonlinux":
		osType = ec2os.AmazonLinuxOS
	case "debian":
		osType = ec2os.DebianOS
	case "redhat":
		osType = ec2os.RedHatOS
	case "suse":
		osType = ec2os.SuseOS
	case "fedora":
		osType = ec2os.FedoraOS
	case "":
		osType = ec2os.UbuntuOS // Default
	default:
		return fmt.Errorf("the os type '%v' is not valid", osTypeStr)
	}

	return WithOS(osType)(p)
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
