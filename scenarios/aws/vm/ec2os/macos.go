package ec2os

import (
	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/resources/aws/ec2"
)

type macOS struct { // nolint:unused
	*os.MacOS
	env aws.Environment
}

func newMacOS(env aws.Environment) *macOS { // nolint:unused
	return &macOS{
		MacOS: os.NewMacOS(),
		env:   env,
	}
}
func (*macOS) GetSSHUser() string { return "ec2-user" } // nolint:unused

func (m *macOS) GetImage(arch os.Architecture) (string, error) { // nolint:unused
	return ec2.SearchAMI(m.env, "628277914472", "amzn-ec2-macos-13.*", m.GetAMIArch(arch))
}

func (*macOS) GetAMIArch(arch os.Architecture) string { // nolint:unused
	switch arch {
	case os.AMD64Arch:
		return "x86_64_mac"
	case os.ARM64Arch:
		return "arm64_mac"
	default:
		panic("Architecture not supported")
	}
}

func (*macOS) GetDefaultInstanceType(arch os.Architecture) string { // nolint:unused
	switch arch {
	case os.AMD64Arch:
		return "mac1.metal"
	case os.ARM64Arch:
		return "mac2.metal"
	default:
		panic("Architecture not supported")
	}
}

func (*macOS) GetTenancy() string { return "host" } // nolint:unused

func (*macOS) GetType() os.Type { // nolint:unused
	return os.OtherType
}
