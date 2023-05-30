package os

import (
	commonos "github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/resources/aws/ec2"
)

type macOS struct { // nolint:unused
	*commonos.MacOS
	env aws.Environment
}

func newMacOS(env aws.Environment) *macOS { // nolint:unused
	return &macOS{
		MacOS: commonos.NewMacOS(),
		env:   env,
	}
}
func (*macOS) GetSSHUser() string { return "ec2-user" } // nolint:unused

func (m *macOS) GetImage(arch commonos.Architecture) (string, error) { // nolint:unused
	return ec2.SearchAMI(m.env, "628277914472", "amzn-ec2-macos-13.*", m.GetAMIArch(arch))
}

func (*macOS) GetAMIArch(arch commonos.Architecture) string { // nolint:unused
	switch arch {
	case commonos.AMD64Arch:
		return "x86_64_mac"
	case commonos.ARM64Arch:
		return "arm64_mac"
	default:
		panic("Architecture not supported")
	}
}

func (*macOS) GetDefaultInstanceType(arch commonos.Architecture) string { // nolint:unused
	switch arch {
	case commonos.AMD64Arch:
		return "mac1.metal"
	case commonos.ARM64Arch:
		return "mac2.metal"
	default:
		panic("Architecture not supported")
	}
}

func (*macOS) GetTenancy() string { return "host" } // nolint:unused

func (*macOS) GetType() commonos.Type { // nolint:unused
	return commonos.OtherType
}
