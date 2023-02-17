package command

import (
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func WaitForCloudInit(ctx *pulumi.Context, runner *Runner) (*remote.Command, error) {
	return runner.Command(
		"wait-cloud-init",
		&Args{
			// `sudo` is required for amazon linux
			Create: pulumi.String("cloud-init status --wait"),
			Sudo:   true,
		})
}
