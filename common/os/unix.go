package os

import (
	"github.com/DataDog/test-infra-definitions/common"
)

type unix struct {
	env common.Environment
}

func (u *unix) GetDefaultInstanceType(arch Architecture) string {
	return getDefaultInstanceType(u.env, arch)
}
func (*unix) GetConfigPath() string { return "/etc/datadog-agent/datadog.yaml" }

func getDefaultInstanceType(env common.Environment, arch Architecture) string {
	switch arch {
	case AMD64Arch:
		return env.DefaultInstanceType()
	case ARM64Arch:
		return env.DefaultARMInstanceType()
	default:
		panic("Architecture not supportede")
	}
}
