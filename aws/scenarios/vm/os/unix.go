package os

import "github.com/DataDog/test-infra-definitions/common/os"

type unix struct{}

func (*unix) GetAMIArch(arch os.Architecture) string { return string(arch) }
func (*unix) GetTenancy() string                     { return "default" }
