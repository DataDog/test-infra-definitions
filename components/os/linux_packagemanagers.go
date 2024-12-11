package os

import (
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func newAptManager(runner *command.RemoteRunner) PackageManager {
	return command.NewGenericPackageManager(runner, "apt", "apt-get install -y", "apt-get update -y",
		pulumi.StringMap{"DEBIAN_FRONTEND": pulumi.String("noninteractive")})
}

func newYumManager(runner *command.RemoteRunner) PackageManager {
	return command.NewGenericPackageManager(runner, "yum", "yum install -y", "", nil)
}

func newDnfManager(runner *command.RemoteRunner) PackageManager {
	return command.NewGenericPackageManager(runner, "dnf", "dnf install -y", "", nil)
}

func newZypperManager(runner *command.RemoteRunner) PackageManager {
	return command.NewGenericPackageManager(runner, "zypper", "zypper -n install", "zypper -n refresh", nil)
}
