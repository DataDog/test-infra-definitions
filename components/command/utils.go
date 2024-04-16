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
			// retry 5 times with exponential backoff as cloud-init may take some time to initialize
			Create: pulumi.String("for i in 1 2 3 4 5; do cloud-init status --wait && break || sleep $((2**$i)); done"),
			Sudo:   true,
		})
}

func WaitForSuccessfulConnection(runner *Runner) (*remote.Command, error) {
	return runner.Command(
		"wait-successful-connection",
		&Args{
			// echo works in shell and powershell
			Create: pulumi.String("echo \"OK\""),
		})
}
