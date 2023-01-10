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
	OS                         OS
	Arch                       os.Architecture
	OptionalAgentInstallParams *agentinstall.Params
	osFactory                  func(os.OSType) (*OS, error)
}

func NewParams[OS os.OS](oses []OS) *Params[OS] {
	params := &Params[OS]{
		osFactory: func(osType os.OSType) (*OS, error) {
			for _, os := range oses {
				if os.GetOSType() == osType {
					return &os, nil
				}
			}
			return nil, fmt.Errorf("%v is not suppported on this environment", osType)
		},
	}

	// By default use Ubuntu
	params.setOS(os.UbuntuOS, os.AMD64Arch)

	return params
}

func (p *Params[OS]) GetInstanceNameOrDefault(defaultName string) string {
	if p.instanceName == "" {
		return defaultName
	}
	return p.instanceName
}

type ParamsGetter[OS os.OS] interface {
	GetCommonParams() *Params[OS]
}

func WithName[OS os.OS, P ParamsGetter[OS]](name string) func(P) error {
	return func(params P) error {
		p := params.GetCommonParams()
		p.instanceName = name
		return nil
	}
}

// WithOS sets the instance type and the AMI.
func WithOS[OS os.OS, P ParamsGetter[OS]](osType os.OSType, arch os.Architecture) func(P) error {
	return func(params P) error {
		p := params.GetCommonParams()
		p.setOS(osType, arch)
		return nil
	}
}

func (p *Params[OS]) setOS(osType os.OSType, arch os.Architecture) error {
	os, err := p.osFactory(osType)
	if err != nil {
		return err
	}
	p.OS = *os
	p.InstanceType = p.OS.GetDefaultInstanceType(arch)
	p.Arch = arch
	p.ImageName, err = p.OS.GetImage(arch)
	if err != nil {
		return fmt.Errorf("cannot find image for %v (%v): %v", osType, arch, err)
	}

	return nil
}

// WithImageName set the name of the Image. `arch` and `osType` must match the AMI requirements.
func WithImageName[OS os.OS, P ParamsGetter[OS]](imageName string, arch os.Architecture, osType os.OSType) func(P) error {
	return func(params P) error {
		p := params.GetCommonParams()
		p.ImageName = imageName
		os, err := p.osFactory(osType)
		if err != nil {
			return err
		}
		p.OS = *os
		p.Arch = arch
		return nil
	}
}

// WithInstanceType set the instance type
func WithInstanceType[OS os.OS, P ParamsGetter[OS]](instanceType string) func(P) error {
	return func(params P) error {
		p := params.GetCommonParams()
		p.InstanceType = instanceType
		return nil
	}
}

// WithUserData set the userdata for the instance. User data contains commands that are run at the startup of the instance.
func WithUserData[OS os.OS, P ParamsGetter[OS]](userData string) func(P) error {
	return func(params P) error {
		p := params.GetCommonParams()
		p.UserData = userData
		return nil
	}
}

// WithHostAgent installs an Agent on this instance. By default use with agentinstall.WithLatest().
func WithHostAgent[OS os.OS, P ParamsGetter[OS]](apiKey string, options ...func(*agentinstall.Params) error) func(P) error {
	return func(params P) error {
		p := params.GetCommonParams()
		var err error
		p.OptionalAgentInstallParams, err = agentinstall.NewParams(apiKey, options...)
		return err
	}
}
