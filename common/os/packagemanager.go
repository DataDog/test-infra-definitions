package os

import (
	"github.com/DataDog/test-infra-definitions/command"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func NewAptManager(runner *command.Runner) command.PackageManager {
	return command.NewGenericPackageManager(runner, "apt", "apt-get install -y", "apt-get update -y",
		pulumi.StringMap{"DEBIAN_FRONTEND": pulumi.String("noninteractive")})
}

func NewBrewManager(runner *command.Runner) command.PackageManager {
	return command.NewGenericPackageManager(runner, "brew", "brew install -y", "brew update -y",
		pulumi.StringMap{"NONINTERACTIVE": pulumi.String("1")})
}
