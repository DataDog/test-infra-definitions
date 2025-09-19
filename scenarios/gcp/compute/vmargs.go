package compute

import (
	"github.com/DataDog/test-infra-definitions/common"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/os"
)

type vmArgs struct {
	osInfo       *os.Descriptor
	instanceType string
	imageName    string
	nestedVirt   bool
}

type VMOption = func(*vmArgs) error

func newParams(options ...VMOption) (*vmArgs, error) {
	vmArgs := &vmArgs{}

	return common.ApplyOption(vmArgs, options)
}

// WithOS sets the OS
// Version defaults to latest
func WithOS(osDesc os.Descriptor) VMOption {
	return WithOSArch(osDesc, osDesc.Architecture)
}

// WithInstanceType sets the VM instance type
// instanceType must be a valid gcp instance type,
// see: https://cloud.google.com/compute/docs/general-purpose-machines
func WithInstancetype(instanceType string) VMOption {
	return func(p *vmArgs) error {
		p.instanceType = instanceType

		return nil
	}
}

// WithArch set the architecture and the operating system.
// Version defaults to latest
func WithOSArch(osDesc os.Descriptor, arch os.Architecture) VMOption {
	return func(p *vmArgs) error {
		p.osInfo = utils.Pointer(osDesc.WithArch(arch))
		return nil
	}
}

// WithNestedVirt sets if the VM allows nested virtualization
// This is useful in case of openshift as it only runs on a VM.
func WithNestedVirt(enabled bool) VMOption {
	return func (p *vmArgs) error {
		p.nestedVirt = enabled
		return nil
	}
}
