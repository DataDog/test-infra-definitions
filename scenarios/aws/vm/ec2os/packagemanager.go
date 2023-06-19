package ec2os

import "github.com/DataDog/test-infra-definitions/components/command"

func newYumManager(runner *command.Runner) command.PackageManager {
	return command.NewGenericPackageManager(runner, "yum", "yum install -y", "", nil)
}

func newDnfManager(runner *command.Runner) command.PackageManager {
	return command.NewGenericPackageManager(runner, "dnf", "dnf install -y", "", nil)
}

func newZypperManager(runner *command.Runner) command.PackageManager {
	return command.NewGenericPackageManager(runner, "zypper", "zypper -n install", "zypper -n refresh", nil)
}
