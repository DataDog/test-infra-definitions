package os

import (
	"github.com/DataDog/test-infra-definitions/aws"
	commonos "github.com/DataDog/test-infra-definitions/common/os"
)

type OS interface {
	commonos.OS
	GetAMIArch(arch commonos.Architecture) string
	GetTenancy() string
}

func GetSupportedOSes(env aws.Environment) []OS {
	return []OS{newWindows(env), newUbuntu(env), newMacOS(env)}
}
