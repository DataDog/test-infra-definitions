package command

import (
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func WaitForCloudInit(ctx *pulumi.Context, runner *Runner) (*remote.Command, error) {
	return runner.Command("wait-cloud-init", pulumi.String("cloud-init status --wait"), nil, nil, nil, false)
}
