package os

import "github.com/DataDog/test-infra-definitions/aws"

type unix struct {
	env aws.Environment
}

func (*unix) GetAMIArch(arch Architecture) string { return string(arch) }
func (u *unix) GetDefaultInstanceType(arch Architecture) string {
	return getDefaultInstanceType(u.env, arch)
}
func (*unix) GetTenancy() string    { return "default" }
func (*unix) GetConfigPath() string { return "/etc/datadog-agent/datadog.yaml" }

func getDefaultInstanceType(env aws.Environment, arch Architecture) string {
	switch arch {
	case AMD64Arch:
		return env.DefaultInstanceType()
	case ARM64Arch:
		return env.DefaultARMInstanceType()
	default:
		panic("Architecture not supportede")
	}
}
