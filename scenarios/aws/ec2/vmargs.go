package ec2

import (
	"github.com/DataDog/test-infra-definitions/common"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/os"
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

type vmArgs struct {
	osInfo          *os.Descriptor
	ami             string
	userData        string
	instanceType    string
	instanceProfile string

	httpTokensRequired bool
}

type VMOption = func(*vmArgs) error

func buildArgs(options ...VMOption) (*vmArgs, error) {
	vmArgs := &vmArgs{}
	return common.ApplyOption(vmArgs, options)
}

// WithOS sets the OS
// Architecture defaults to os.AMD64Arch
// Version defaults to latest
func WithOS(osDesc os.Descriptor) VMOption {
	return WithOSArch(osDesc, os.AMD64Arch)
}

// WithArch set the architecture and the operating system.
// Version defaults to latest
func WithOSArch(osDesc os.Descriptor, arch os.Architecture) VMOption {
	return func(p *vmArgs) error {
		p.osInfo = utils.Pointer(osDesc.WithArch(arch))
		return nil
	}
}

// WithAMI sets the AMI directly, skipping resolve process. `supportedOS` and `arch` must match the AMI requirements.
func WithAMI(ami string, osDesc os.Descriptor, arch os.Architecture) VMOption {
	return func(p *vmArgs) error {
		p.osInfo = utils.Pointer(osDesc.WithArch(arch))
		p.ami = ami
		return nil
	}
}

// WithInstanceType set the instance type
func WithInstanceType(instanceType string) VMOption {
	return func(p *vmArgs) error {
		p.instanceType = instanceType
		return nil
	}
}

// WithUserData set the userdata for the instance. User data contains commands that are run at the startup of the instance.
func WithUserData(userData string) VMOption {
	return func(p *vmArgs) error {
		p.userData = userData
		return nil
	}
}

func WithInstanceProfile(instanceProfile string) VMOption {
	return func(p *vmArgs) error {
		p.instanceProfile = instanceProfile
		return nil
	}
}

func WithHTTPTokensRequired() VMOption {
	return func(p *vmArgs) error {
		p.httpTokensRequired = true
		return nil
	}
}
