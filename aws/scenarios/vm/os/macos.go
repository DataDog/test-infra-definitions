package os

import (
	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/aws/ec2/ec2"
)

type macOS struct {
	env aws.Environment
}

func (*macOS) GetSSHUser() string { return "ec2-user" }

func (m *macOS) GetAMI(arch Architecture) (string, error) {
	return ec2.SearchAMI(m.env, "628277914472", "amzn-ec2-macos-13.*", m.GetAMIArch(arch))
}

func (*macOS) GetAMIArch(arch Architecture) string {
	switch arch {
	case AMD64Arch:
		return "x86_64_mac"
	case ARM64Arch:
		return "arm64_mac"
	default:
		panic("Architecture not supported")
	}
}

func (*macOS) GetDefaultInstanceType(arch Architecture) string {
	switch arch {
	case AMD64Arch:
		return "mac1.metal"
	case ARM64Arch:
		return "mac2.metal"
	default:
		panic("Architecture not supported")
	}
}

func (*macOS) GetTenancy() string { return "host" }

func (*macOS) GetServiceManager() *serviceManager {
	return &serviceManager{restartCmd: []string{"launchctl stop com.datadoghq.agent", "launchctl start com.datadoghq.agent"}}
}

func (*macOS) GetConfigPath() string { return "~/.datadog-agent/datadog.yaml" }

func (*macOS) GetOSType() OSType { return MacOS }
