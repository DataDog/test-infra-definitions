package localdocker

import (
	"github.com/DataDog/test-infra-definitions/components"
	"github.com/DataDog/test-infra-definitions/components/command"
	componentsos "github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/components/remote"
	"github.com/DataDog/test-infra-definitions/resources/local"
	localdocker "github.com/DataDog/test-infra-definitions/resources/local/docker"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// NewVM creates an localdocker Ubuntu VM Instance and returns a Remote component.
// Without any parameter it creates an Ubuntu VM on AMD64 architecture.
func NewVM(e local.Environment, name string) (*remote.Host, error) {
	// Create the EC2 instance
	return components.NewComponent(&e, e.Namer.ResourceName(name), func(c *remote.Host) error {
		vmArgs := &localdocker.VMArgs{
			Name: name,
		}

		// Create the EC2 instance
		address, user, port, err := localdocker.NewInstance(e, *vmArgs, pulumi.Parent(c))
		if err != nil {
			return err
		}

		// Create connection
		conn, err := remote.NewConnection(
			address,
			user,
			remote.WithPort(port),
		)
		if err != nil {
			return err
		}
		return remote.InitHost(&e, conn.ToConnectionOutput(), componentsos.Ubuntu2204, user, pulumi.String("").ToStringOutput(), command.WaitForSuccessfulConnection, c)
	})
}
