package os

import (
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func newBrewManager(runner *command.RemoteRunner) PackageManager {
	return command.NewGenericPackageManager(runner, "brew", "brew install -y", "brew update -y",
		pulumi.StringMap{"NONINTERACTIVE": pulumi.String("1")})
}
