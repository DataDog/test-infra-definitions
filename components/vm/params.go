package vm

import (
	"fmt"
	"reflect"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/components/os"
)

type Params[OS os.OS] struct {
	InstanceName string
	ImageName    string
	InstanceType string
	UserData     string
	OS           OS
	Arch         os.Architecture
	commonEnv    *config.CommonEnvironment
}

func NewParams[OS os.OS](commonEnv *config.CommonEnvironment) (*Params[OS], error) {
	params := &Params[OS]{
		commonEnv:    commonEnv,
		InstanceName: "vm",
	}

	return params, nil
}

func (p *Params[OS]) SetName(name string) error {
	p.InstanceName = name
	return nil
}

// SetOS sets the OS. This function also set the instance type and the AMI.
func (p *Params[OS]) SetOS(o OS) error {
	return p.SetArch(o, os.AMD64Arch)
}

// SetArch set the architecture and the operating system.
func (p *Params[OS]) SetArch(os OS, arch os.Architecture) error {
	var err error
	p.ImageName, err = os.GetImage(arch)
	if err != nil {
		return fmt.Errorf("cannot find image for %v (%v): %v", reflect.TypeOf(os), arch, err)
	}
	p.OS = os
	p.InstanceType = p.OS.GetDefaultInstanceType(arch)
	p.Arch = arch

	return nil
}

// SetImageName set the name of the Image. `arch` and `osType` must match the AMI requirements.
func (p *Params[OS]) SetImageName(imageName string, arch os.Architecture, os OS) error {
	p.ImageName = imageName
	p.OS = os
	p.Arch = arch
	return nil
}

// SetInstanceType set the instance type
func (p *Params[OS]) SetInstanceType(instanceType string) error {
	p.InstanceType = instanceType
	return nil

}

// SetUserData set the userdata for the instance. User data contains commands that are run at the startup of the instance.
func (p *Params[OS]) SetUserData(userData string) error {
	p.UserData = userData
	return nil
}
