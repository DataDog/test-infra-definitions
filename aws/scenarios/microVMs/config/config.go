package config

import (
	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/aws/ec2"
	sdkconfig "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

const (
	ddMicroVMNamespace            = "microvm"
	ddMicroVMX86LibvirtSSHKeyFile = "libvirtSSHKeyFileX86"
	ddMicroVMArmLibvirtSSHKeyFile = "libvirtSSHKeyFileArm"

	DDMicroVMProvisionEC2Instance = "provision"
	DDMicroVMX86AmiID             = "x86AmiID"
	DDMicroVMArm64AmiID           = "arm64AmiID"
	DDMicroVMConfigFile           = "microVMConfigFile"
	DDMicroVMWorkingDirectory     = "workingDir"
	DDMicroVMShutdownPeriod       = "shutdownPeriod"
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
