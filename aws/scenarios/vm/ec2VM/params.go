package ec2vm

import (
	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/aws/scenarios/vm/os"
	"github.com/DataDog/test-infra-definitions/common"
	"github.com/DataDog/test-infra-definitions/common/vm"
)

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

	if err := WithOS(os.UbuntuOS)(params); err != nil {
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

// WithOS sets the instance type and the AMI.
var WithOS = vm.WithOS[os.OS, os.Type, *Params]

// WithImageName set the name of the Image. `arch` and `osType` must match the AMI requirements.
var WithImageName = vm.WithImageName[os.OS, os.Type, *Params]

// WithArch set the architecture and the operating system.
var WithArch = vm.WithArch[os.OS, os.Type, *Params]

// WithInstanceType set the instance type.
var WithInstanceType = vm.WithInstanceType[os.OS, os.Type, *Params]

// WithUserData set the userdata for the EC2 instance. User data contains commands that are run at the startup of the instance.
var WithUserData = vm.WithUserData[os.OS, os.Type, *Params]

// WithName set the VM name
var WithName = vm.WithName[os.OS, os.Type, *Params]
