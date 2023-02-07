package vm

import (
	"github.com/DataDog/test-infra-definitions/azure"
	"github.com/DataDog/test-infra-definitions/azure/compute/os"
	"github.com/DataDog/test-infra-definitions/common"
	commonos "github.com/DataDog/test-infra-definitions/common/os"
	"github.com/DataDog/test-infra-definitions/common/vm"
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
	if err := WithOS(os.UbuntuOS, commonos.AMD64Arch)(params); err != nil {
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

// WithOS sets the instance type and the AMI.
var WithOS = vm.WithOS[commonos.OS, os.Type, *Params]

// WithImageName set the name of the Image. `arch` and `osType` must match the AMI requirements.
var WithImageName = vm.WithImageName[commonos.OS, os.Type, *Params]

// WithInstanceType set the instance type
var WithInstanceType = vm.WithInstanceType[commonos.OS, os.Type, *Params]

// WithUserData set the userdata for the EC2 instance. User data contains commands that are run at the startup of the instance.
var WithUserData = vm.WithUserData[commonos.OS, os.Type, *Params]

// WithName set the VM name
var WithName = vm.WithName[commonos.OS, os.Type, *Params]
