package vm

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/common/agentinstall"
	"github.com/DataDog/test-infra-definitions/common/os"
)

type Params[OS os.OS] struct {
	instanceName               string
	ImageName                  string
	InstanceType               string
	UserData                   string
	Os                         OS
	Arch                       os.Architecture
	OptionalAgentInstallParams *agentinstall.Params
	osFactory                  func(os.OSType) OS
}

func NewParams[OS os.OS](osFactory func(os.OSType) OS) *Params[OS] {
	params := &Params[OS]{
		osFactory: osFactory,
	}

	// By default use Ubuntu
	params.SetOS(os.UbuntuOS, os.AMD64Arch)
	return params
}

func (p *Params[OS]) GetInstanceNameOrDefault(defaultName string) string {
	if p.instanceName == "" {
		return defaultName
	}
	return p.instanceName
}

func (p *Params[OS]) SetName(name string) error {
	p.instanceName = name
	return nil
}

// SetOS sets the instance type and the AMI.
func (p *Params[OS]) SetOS(osType os.OSType, arch os.Architecture) error {
	var err error
	var os = p.osFactory(osType)

	p.InstanceType = os.GetDefaultInstanceType(arch)
	p.Arch = arch
	p.Os = os
	p.ImageName, err = os.GetImage(arch)
	if err != nil {
		return fmt.Errorf("cannot find image for %v (%v): %v", osType, arch, err)
	}

	return nil
}

// SetImageName set the name of the Image. `arch` and `osType` must match the AMI requirements.
func (p *Params[OS]) SetImage(imageName string, arch os.Architecture, osType os.OSType) error {
	p.ImageName = imageName
	p.Os = p.osFactory(osType)
	p.Arch = arch
	return nil
}

// SetInstanceType set the instance type
func (p *Params[OS]) SetInstanceType(instanceType string) error {
	p.InstanceType = instanceType
	return nil
}

// SetUserData set the userdata for the EC2 instance. User data contains commands that are run at the startup of the instance.
func (p *Params[OS]) SetUserData(userData string) error {
	p.UserData = userData
	return nil
}

// SetHostAgent installs an Agent on this EC2 instance. By default use with agentinstall.WithLatest().
func (p *Params[OS]) SetHostAgent(apiKey string, options ...func(*agentinstall.Params) error) error {
	var err error
	p.OptionalAgentInstallParams, err = agentinstall.NewParams(apiKey, options...)
	return err
}
