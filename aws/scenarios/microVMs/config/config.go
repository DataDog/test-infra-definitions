package config

import (
	"errors"
	"fmt"

	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/aws/ec2/ec2"
	sdkconfig "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

const (
	ddMicroVMNamespace            = "microvm"
	ddMicroVMX86LibvirtSSHKeyFile = "libvirtSSHKeyFileX86"
	ddMicroVMArmLibvirtSSHKeyFile = "libvirtSSHKeyFileArm"

	DDMicroVMProvisionEC2Instance = "provision"
	DDMicroVMAMIID                = "amiID"
	DDMicroVMConfigFile           = "microVMConfigFile"
	DDMicroVMWorkingDirectory     = "workingDir"
)

var SSHKeyConfigNames = map[string]string{
	ec2.AMD64Arch: ddMicroVMX86LibvirtSSHKeyFile,
	ec2.ARM64Arch: ddMicroVMArmLibvirtSSHKeyFile,
}

type DDMicroVMConfig struct {
	MicroVMConfig *sdkconfig.Config
	aws.Environment
}

func NewMicroVMConfig(e aws.Environment) DDMicroVMConfig {
	return DDMicroVMConfig{
		sdkconfig.New(e.Ctx, ddMicroVMNamespace),
		e,
	}
}

func (e *DDMicroVMConfig) GetStringWithDefault(config *sdkconfig.Config, paramName string, defaultValue string) string {
	val, err := config.Try(paramName)
	if err == nil {
		return val
	}

	if !errors.Is(err, sdkconfig.ErrMissingVar) {
		e.Ctx.Log.Error(fmt.Sprintf("Parameter %s not parsable, err: %v, will use default value: %v", paramName, err, defaultValue), nil)
	}

	return defaultValue
}

func (e *DDMicroVMConfig) GetIntWithDefault(config *sdkconfig.Config, paramName string, defaultValue int) int {
	val, err := config.TryInt(paramName)
	if err == nil {
		return val
	}

	if !errors.Is(err, sdkconfig.ErrMissingVar) {
		e.Ctx.Log.Error(fmt.Sprintf("Parameter %s not parsable, err: %v, will use default value: %v", paramName, err, defaultValue), nil)
	}

	return defaultValue

}

func (e *DDMicroVMConfig) GetBoolWithDefault(config *sdkconfig.Config, paramName string, defaultValue bool) bool {
	val, err := config.TryBool(paramName)
	if err == nil {
		return val
	}

	if !errors.Is(err, sdkconfig.ErrMissingVar) {
		e.Ctx.Log.Error(fmt.Sprintf("Parameter %s not parsable, err: %v, will use default value: %v", paramName, err, defaultValue), nil)
	}

	return defaultValue
}
