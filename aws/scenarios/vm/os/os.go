package os

import (
	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/common/os"
)

type OS interface {
	os.OS
	GetAMIArch(arch os.Architecture) string
	GetTenancy() string
}

func GetOSes(env aws.Environment) []OS {
	return []OS{newWindows(env), newUbuntu(env), newMacOS(env)}
}
