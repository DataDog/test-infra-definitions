package ec2vm

import (
	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/aws/scenarios/vm/os"
	"github.com/DataDog/test-infra-definitions/common"
	"github.com/DataDog/test-infra-definitions/common/agentinstall"
	commonos "github.com/DataDog/test-infra-definitions/common/os"
	"github.com/DataDog/test-infra-definitions/common/vm"
)

type Params struct {
	common  *vm.Params[os.OS]
	keyPair string
}

func newParams(env aws.Environment, options ...func(*Params) error) (*Params, error) {
	params := &Params{
		keyPair: env.DefaultKeyPairName(),
		common:  vm.NewParams(func(osType commonos.OSType) os.OS { return os.GetOS(env, osType) }),
	}

	return common.ApplyOption(params, options)
}

// WithOS sets the instance type and the AMI.
func WithOS(osType commonos.OSType, arch commonos.Architecture) func(*Params) error {
	return func(p *Params) error { return p.common.SetOS(osType, arch) }
}

// WithImageName set the name of the Image. `arch` and `osType` must match the AMI requirements.
func WithImage(imageName string, arch commonos.Architecture, osType commonos.OSType) func(*Params) error {
	return func(p *Params) error { return p.common.SetImage(imageName, arch, osType) }
}

// WithInstanceType set the instance type
func WithInstanceType(instanceType string) func(*Params) error {
	return func(p *Params) error { return p.common.SetInstanceType(instanceType) }
}

// WithUserData set the userdata for the EC2 instance. User data contains commands that are run at the startup of the instance.
func WithUserData(userData string) func(*Params) error {
	return func(p *Params) error { return p.common.SetUserData(userData) }
}

// WithHostAgent installs an Agent on this EC2 instance. By default use with agentinstall.WithLatest().
func WithHostAgent(apiKey string, options ...func(*agentinstall.Params) error) func(*Params) error {
	return func(p *Params) error { return p.common.SetHostAgent(apiKey, options...) }
}

// WithName set the VM name
func WithName(name string) func(*Params) error {
	return func(p *Params) error { return p.common.SetName(name) }
}
