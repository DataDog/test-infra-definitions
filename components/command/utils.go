package command

import (
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type ReadyFunc func(*Runner) (*remote.Command, error)

func WaitForCloudInit(runner *Runner) (*remote.Command, error) {
	return runner.Command(
		"wait-cloud-init",
		&Args{
			// `sudo` is required for amazon linux
			Create: pulumi.String("cloud-init status --wait"),
			Sudo:   true,
		})
}

func WaitUntilSuccess(runner *Runner) (*remote.Command, error) {
	return runner.Command(
		"wait-until-success",
		&Args{
			// echo works in shell and powershell
			Create: pulumi.String("echo \"OK\""),
		})
}
