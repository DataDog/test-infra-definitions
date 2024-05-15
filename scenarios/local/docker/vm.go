package docker

import (
	"github.com/DataDog/test-infra-definitions/components"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/components/remote"
	"github.com/DataDog/test-infra-definitions/resources/local/docker"

	premote "github.com/pulumi/pulumi-command/sdk/go/command/remote"
	pdocker "github.com/pulumi/pulumi-docker/sdk/v4/go/docker"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func NewVM(e docker.Environment, name string) (*remote.Host, error) {

	return components.NewComponent(&e, e.Namer.ResourceName(name), func(comp *remote.Host) error {
		instanceArgs := docker.VMArgs{
			Name: name,
		}
		// Create the Docker instance
		agentHost, err := docker.NewInstance(e, instanceArgs, pulumi.Parent(comp))
		if err != nil {
			return err
		}

		// Get SSH port for Agent in container
		sshPort := agentHost.Ports.Index(pulumi.Int(0)).ApplyT(func(v pdocker.ContainerPort) pulumi.Float64PtrOutput {
			m := float64(*v.External)
			return pulumi.Float64Ptr(m).ToFloat64PtrOutput()
		}).(pulumi.Float64PtrOutput)

		conn := &premote.ConnectionArgs{
			Host:           pulumi.String("localhost"),
			Port:           sshPort,
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
