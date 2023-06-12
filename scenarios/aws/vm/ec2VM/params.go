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

	amiID := env.InfraOSAmiId()
	if amiID != "" {
		osType, err := params.getOSType()
		if err != nil {
			return nil, err
		}
		WithImageName(amiID, commonos.Architecture(env.InfraOSArchitecture()), osType)(params)
	} else if err := params.useDefaultOS(); err != nil { // Can be overrided later if the caller uses WithOS.
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

func (p *Params) getOSType() (os.Type, error) {
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
		return os.UbuntuOS, fmt.Errorf("the os type '%v' is not valid", osTypeStr)
	}
	return osType, nil

}

func (p *Params) useDefaultOS() error {

	osType, err := p.getOSType()
	if err != nil {
		return err
	}

	return WithOS(osType)(p)
}

// WithOS sets the instance type and the AMI. See [vm.WithOS] for parameters.
var WithOS = vm.WithOS[os.OS, os.Type, *Params]

// WithImageName set the name of the Image. `arch` and `osType` must match the AMI requirements. See [vm.WithImageName] for parameters.
var WithImageName = vm.WithImageName[os.OS, os.Type, *Params]

// WithArch set the architecture and the operating system. See [vm.WithArch] for parameters.
var WithArch = vm.WithArch[os.OS, os.Type, *Params]

// WithInstanceType set the instance type. See [vm.WithInstanceType] for parameters.
var WithInstanceType = vm.WithInstanceType[os.OS, os.Type, *Params]

// WithUserData set the userdata for the EC2 instance. User data contains commands that are run at the startup of the instance. See [vm.WithUserData] for parameters.
var WithUserData = vm.WithUserData[os.OS, os.Type, *Params]

// WithName set the VM name. See [vm.WithName] for parameters.
var WithName = vm.WithName[os.OS, os.Type, *Params]
