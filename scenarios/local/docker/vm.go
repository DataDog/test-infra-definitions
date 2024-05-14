package docker

import (
	"github.com/DataDog/test-infra-definitions/components"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/components/remote"
	"github.com/DataDog/test-infra-definitions/resources/local/docker"

	premote "github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func NewVM(e docker.Environment, name string) (*remote.Host, error) {

	return components.NewComponent(&e, e.Namer.ResourceName(name), func(comp *remote.Host) error {
		instanceArgs := docker.VMArgs{
			Name: name,
		}
		// Create the Docker instance
		_, err := docker.NewInstance(e, instanceArgs, pulumi.Parent(comp))
		if err != nil {
			return err
		}

		conn := &premote.ConnectionArgs{
			Host:           pulumi.String("localhost"),
			Port:           pulumi.Float64Ptr(3333), // TODO: make dynamic
			Password:       pulumi.String("root123"),
			User:           pulumi.String("root"),
			PerDialTimeout: pulumi.IntPtr(5),
			DialErrorLimit: pulumi.IntPtr(100),
		}

		return remote.InitHost(&e,
			conn.ToConnectionOutputWithContext(e.Ctx().Context()),
			os.Ubuntu2204,
			"root",
			command.WaitForSuccessfulConnection,
			comp,
		)
	})
}
