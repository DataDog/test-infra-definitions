package os

import commonos "github.com/DataDog/test-infra-definitions/components/os"

type unix struct{}

func (*unix) GetAMIArch(arch commonos.Architecture) string { return string(arch) }
func (*unix) GetTenancy() string                           { return "default" }
