package ec2vm

import (
	"fmt"
	"strings"

	"github.com/DataDog/test-infra-definitions/common"
	commonos "github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/components/vm"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/vm/os"
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
	common *vm.Params[os.OS]
}

func newParams(env aws.Environment, options ...func(*Params) error) (*Params, error) {
	commonParams, err := vm.NewParams[os.OS](env.CommonEnvironment)
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

func (p *Params) GetCommonParams() *vm.Params[os.OS] {
	return p.common
}

func (p *Params) GetOS(osType os.Type) (os.OS, error) {
	return os.GetOS(p.env, osType)
}

func (p *Params) useDefaultOS() error {
	var osType os.Type

	osTypeStr := strings.ToLower(p.env.InfraOSFamily())
	switch osTypeStr {
	case "windows":
		osType = os.WindowsOS
	case "ubuntu":
		osType = os.UbuntuOS
	case "amazonlinux":
		osType = os.AmazonLinuxOS
	case "debian":
		osType = os.DebianOS
	case "redhat":
		osType = os.RedHatOS
	case "suse":
		osType = os.SuseOS
	case "fedora":
		osType = os.FedoraOS
	case "":
		osType = os.UbuntuOS // Default
	default:
		return fmt.Errorf("the os type '%v' is not valid", osTypeStr)
	}

	return WithOS(osType)(p)
}

// WithOS sets the OS. This function also set the instance type and the AMI.
func WithOS(osType os.Type) func(*Params) error {
	return vm.WithOS[os.OS, os.Type, *Params](osType)
}

// WithImageName set the name of the Image. `arch` and `osType` must match the AMI requirements.
func WithImageName(imageName string, arch commonos.Architecture, osType os.Type) func(*Params) error {
	return vm.WithImageName[os.OS, os.Type, *Params](imageName, arch, osType)
}

// WithArch set the architecture and the operating system.
func WithArch(osType os.Type, arch commonos.Architecture) func(*Params) error {
	return vm.WithArch[os.OS, os.Type, *Params](osType, arch)
}

// WithInstanceType set the instance type
func WithInstanceType(instanceType string) func(*Params) error {
	return vm.WithInstanceType[os.OS, os.Type, *Params](instanceType)
}

// WithUserData set the userdata for the instance. User data contains commands that are run at the startup of the instance.
func WithUserData(userData string) func(*Params) error {
	return vm.WithUserData[os.OS, os.Type, *Params](userData)
}

// WithName set the name of the instance
func WithName(name string) func(*Params) error {
	return vm.WithName[os.OS, os.Type, *Params](name)
}
