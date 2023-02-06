package vm

import (
	"fmt"
	"reflect"

	"github.com/DataDog/test-infra-definitions/common/agentinstall"
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/os"
)

type Params[OS os.OS] struct {
	InstanceName               string
	ImageName                  string
	InstanceType               string
	UserData                   string
	OS                         OS
	Arch                       os.Architecture
	OptionalAgentInstallParams *agentinstall.Params
	commonEnv                  *config.CommonEnvironment
}

func NewParams[OS os.OS](commonEnv *config.CommonEnvironment) (*Params[OS], error) {
	params := &Params[OS]{
		commonEnv:    commonEnv,
		InstanceName: "vm",
	}

	if commonEnv.AgentDeploy() {
		if err := params.setAgentInstallParams(); err != nil {
			return nil, err
		}
	}

	return params, nil
}

type ParamsGetter[OS os.OS, T any] interface {
	GetCommonParams() *Params[OS]
	GetOS(osType T) (OS, error)
}

func WithName[OS os.OS, T any, P ParamsGetter[OS, T]](name string) func(P) error {
	return func(params P) error {
		p := params.GetCommonParams()
		p.InstanceName = name
		return nil
	}
}

// WithOS sets the instance type and the AMI.
func WithOS[OS os.OS, T any, P ParamsGetter[OS, T]](osType T, arch os.Architecture) func(P) error {
	return func(params P) error {
		p := params.GetCommonParams()
		os, err := params.GetOS(osType)
		if err != nil {
			return err
		}
		return p.setOS(os, arch)
	}
}

func (p *Params[OS]) setOS(os OS, arch os.Architecture) error {
	p.OS = os
	p.InstanceType = p.OS.GetDefaultInstanceType(arch)
	p.Arch = arch
	var err error
	p.ImageName, err = p.OS.GetImage(arch)
	if err != nil {
		return fmt.Errorf("cannot find image for %v (%v): %v", reflect.TypeOf(os), arch, err)
	}

	return nil
}

// WithImageName set the name of the Image. `arch` and `osType` must match the AMI requirements.
func WithImageName[OS os.OS, T any, P ParamsGetter[OS, T]](imageName string, arch os.Architecture, osType T) func(P) error {
	return func(params P) error {
		p := params.GetCommonParams()
		p.ImageName = imageName
		os, err := params.GetOS(osType)
		if err != nil {
			return err
		}
		p.OS = os
		p.Arch = arch
		return nil
	}
}

// WithInstanceType set the instance type
func WithInstanceType[OS os.OS, T any, P ParamsGetter[OS, T]](instanceType string) func(P) error {
	return func(params P) error {
		p := params.GetCommonParams()
		p.InstanceType = instanceType
		return nil
	}
}

// WithUserData set the userdata for the instance. User data contains commands that are run at the startup of the instance.
func WithUserData[OS os.OS, T any, P ParamsGetter[OS, T]](userData string) func(P) error {
	return func(params P) error {
		p := params.GetCommonParams()
		p.UserData = userData
		return nil
	}
}

// WithHostAgent installs an Agent on this instance. By default use with agentinstall.WithLatest().
func WithHostAgent[OS os.OS, T any, P ParamsGetter[OS, T]](options ...func(*agentinstall.Params) error) func(P) error {
	return func(params P) error {
		p := params.GetCommonParams()
		return p.setAgentInstallParams(options...)
	}
}

func (p *Params[OS]) setAgentInstallParams(options ...func(*agentinstall.Params) error) error {
	var err error
	p.OptionalAgentInstallParams, err = agentinstall.NewParams(p.commonEnv, options...)
	return err
}
