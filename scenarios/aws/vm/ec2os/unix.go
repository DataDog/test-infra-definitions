package ec2os

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/components/os"
)

type unix struct {
	env config.Environment
}

func newUnix(env config.Environment) *unix {
	return &unix{env: env}
}

func (*unix) GetAMIArch(arch os.Architecture) string { return string(arch) }
func (*unix) GetTenancy() string                     { return "default" }
func (u *unix) GetDefaultInstanceType(arch os.Architecture) string {
	return os.GetDefaultInstanceType(u.env, arch)
}
