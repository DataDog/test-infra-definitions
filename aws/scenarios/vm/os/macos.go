package os

import (
	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/aws/ec2/ec2"
	commonos "github.com/DataDog/test-infra-definitions/common/os"
)

type macOS struct {
	*commonos.MacOS
	env aws.Environment
}

func newMacOS(env aws.Environment) *macOS {
	return &macOS{
		MacOS: commonos.NewMacOS(),
		env:   env,
	}
}
func (*macOS) GetSSHUser() string { return "ec2-user" }

func (m *macOS) GetImage(arch commonos.Architecture) (string, error) {
	return ec2.SearchAMI(m.env, "628277914472", "amzn-ec2-macos-13.*", m.GetAMIArch(arch))
}

func (*macOS) GetAMIArch(arch commonos.Architecture) string {
	switch arch {
	case commonos.AMD64Arch:
		return "x86_64_mac"
	case commonos.ARM64Arch:
		return "arm64_mac"
	default:
		panic("Architecture not supported")
	}
}

func (*macOS) GetDefaultInstanceType(arch commonos.Architecture) string {
	switch arch {
	case commonos.AMD64Arch:
		return "mac1.metal"
	case commonos.ARM64Arch:
		return "mac2.metal"
	default:
		panic("Architecture not supported")
	}
}

func (*macOS) GetTenancy() string { return "host" }

func (*macOS) GetType() commonos.Type {
	return commonos.OtherType
}
