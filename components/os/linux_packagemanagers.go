package os

import (
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type (
	PackageManagerType string
)

const (
	Apt PackageManagerType = "apt"
	Yum PackageManagerType = "yum"
	Dnf PackageManagerType = "dnf"
)

func (pm PackageManagerType) String() string {
	return string(pm)
}

func newAptManager(runner command.Runner) PackageManager {
	return NewGenericPackageManager(runner, Apt, "apt-get install -y", "apt-get update -y", "apt-get remove -y",
		pulumi.StringMap{"DEBIAN_FRONTEND": pulumi.String("noninteractive")})
}

func newYumManager(runner command.Runner) PackageManager {
	return NewGenericPackageManager(runner, Yum, "yum install -y", "", "yum remove -y", nil)
}

func newDnfManager(runner command.Runner) PackageManager {
	return NewGenericPackageManager(runner, Dnf, "dnf install -y", "", "dnf remove -y", nil)
}
