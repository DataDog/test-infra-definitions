package os

import (
	"github.com/DataDog/test-infra-definitions/common/config"
)

func GetDefaultInstanceType(env config.Environment, arch Architecture) string {
	switch arch {
	case AMD64Arch:
		return env.DefaultInstanceType()
	case ARM64Arch:
		return env.DefaultARMInstanceType()
	default:
		panic("Architecture not supported")
	}
}
