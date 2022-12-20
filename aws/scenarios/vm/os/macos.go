package os

import (
	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/aws/ec2/ec2"
	"github.com/DataDog/test-infra-definitions/common/os"
)

type macOS struct {
	*os.MacOS
	env aws.Environment
}

func newMacOS(env aws.Environment) *macOS {
	return &macOS{
		MacOS: os.NewMacOS(),
		env:   env,
	}
}
func (*macOS) GetSSHUser() string { return "ec2-user" }

func (m *macOS) GetImage(arch os.Architecture) (string, error) {
	return ec2.SearchAMI(m.env, "628277914472", "amzn-ec2-macos-13.*", m.GetAMIArch(arch))
}

func (*macOS) GetAMIArch(arch os.Architecture) string {
	switch arch {
	case os.AMD64Arch:
		return "x86_64_mac"
	case os.ARM64Arch:
		return "arm64_mac"
	default:
		panic("Architecture not supported")
	}
}

func (*macOS) GetDefaultInstanceType(arch os.Architecture) string {
	switch arch {
	case os.AMD64Arch:
		return "mac1.metal"
	case os.ARM64Arch:
		return "mac2.metal"
	default:
		panic("Architecture not supported")
	}
}

func (*macOS) GetTenancy() string { return "host" }
