package vm

import (
	"fmt"
	"reflect"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/os"
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

// WithOS sets the OS. This function also set the instance type and the AMI.
func WithOS[OS os.OS, T any, P ParamsGetter[OS, T]](osType T) func(P) error {
	return WithArch[OS, T, P](osType, os.AMD64Arch)
}

// WithArch set the architecture and the operating system.
func WithArch[OS os.OS, T any, P ParamsGetter[OS, T]](osType T, arch os.Architecture) func(P) error {
	return func(params P) error {
		p := params.GetCommonParams()
		os, err := params.GetOS(osType)
		if err != nil {
			return err
		}
		p.ImageName, err = os.GetImage(arch)
		if err != nil {
			return fmt.Errorf("cannot find image for %v (%v): %v", reflect.TypeOf(os), arch, err)
		}
		p.OS = os
		p.InstanceType = p.OS.GetDefaultInstanceType(arch)
		p.Arch = arch

		return nil
	}
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
