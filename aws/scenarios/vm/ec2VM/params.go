package ec2vm

import (
	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/aws/scenarios/vm/os"
	"github.com/DataDog/test-infra-definitions/common"
	"github.com/DataDog/test-infra-definitions/common/vm"
)

type Params struct {
	common  *vm.Params[os.OS]
	keyPair string
}

func newParams(env aws.Environment, options ...func(*Params) error) (*Params, error) {
	params := &Params{
		keyPair: env.DefaultKeyPairName(),
		common:  vm.NewParams(os.GetOSes(env)),
	}

	return common.ApplyOption(params, options)
}

func (p *Params) GetCommonParams() *vm.Params[os.OS] {
	return p.common
}

// WithOS sets the instance type and the AMI.
var WithOS = vm.WithOS[os.OS, *Params]

// WithImageName set the name of the Image. `arch` and `osType` must match the AMI requirements.
var WithImageName = vm.WithImageName[os.OS, *Params]

// WithInstanceType set the instance type
var WithInstanceType = vm.WithInstanceType[os.OS, *Params]

// WithUserData set the userdata for the EC2 instance. User data contains commands that are run at the startup of the instance.
var WithUserData = vm.WithUserData[os.OS, *Params]

// WithHostAgent installs an Agent on this EC2 instance. By default use with agentinstall.WithLatest().
var WithHostAgent = vm.WithHostAgent[os.OS, *Params]

// WithName set the VM name
var WithName = vm.WithName[os.OS, *Params]
