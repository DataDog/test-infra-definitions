package vm

import (
	"github.com/DataDog/test-infra-definitions/azure"
	"github.com/DataDog/test-infra-definitions/azure/compute/os"
	"github.com/DataDog/test-infra-definitions/common"
	commonos "github.com/DataDog/test-infra-definitions/common/os"
	"github.com/DataDog/test-infra-definitions/common/vm"
)

type Params struct {
	common *vm.Params[commonos.OS]
}

func newParams(env azure.Environment, options ...func(*Params) error) (*Params, error) {
	commonParams, err := vm.NewParams(env.CommonEnvironment, os.GetOSes(env))
	if err != nil {
		return nil, err
	}
	params := &Params{
		common: commonParams,
	}

	return common.ApplyOption(params, options)
}

func (p *Params) GetCommonParams() *vm.Params[commonos.OS] {
	return p.common
}

// WithOS sets the instance type and the AMI.
var WithOS = vm.WithOS[commonos.OS, *Params]

// WithImageName set the name of the Image. `arch` and `osType` must match the AMI requirements.
var WithImageName = vm.WithImageName[commonos.OS, *Params]

// WithInstanceType set the instance type
var WithInstanceType = vm.WithInstanceType[commonos.OS, *Params]

// WithUserData set the userdata for the EC2 instance. User data contains commands that are run at the startup of the instance.
var WithUserData = vm.WithUserData[commonos.OS, *Params]

// WithHostAgent installs an Agent on this EC2 instance. By default use with agentinstall.WithLatest().
var WithHostAgent = vm.WithHostAgent[commonos.OS, *Params]

// WithName set the VM name
var WithName = vm.WithName[commonos.OS, *Params]
