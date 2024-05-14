package docker

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/components"
	"github.com/DataDog/test-infra-definitions/components/remote"
	"github.com/DataDog/test-infra-definitions/resources/local/docker"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func NewVM(e docker.Environment, name string) (*remote.Host, error) {

	return components.NewComponent(&e, e.Namer.ResourceName(name), func(c *remote.Host) error {
		instanceArgs := docker.VMArgs{
			Name: name,
		}
		// Create the Docker instance
		instance, err := docker.NewInstance(e, instanceArgs, pulumi.Parent(c))
		if err != nil {
			return err
		}

		fmt.Println("Docker instance created", instance)
		return nil
	})
}
