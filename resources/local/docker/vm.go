package docker

import (
	"fmt"

	config "github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/remote"
	"github.com/pulumi/pulumi-docker/sdk/v4/go/docker"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type VMArgs struct {
	Name string
	// Attributes you need when you will actually create the VM
}

func NewInstance(e Environment, args VMArgs, opts ...pulumi.ResourceOption) (*remote.Host, error) {
	hostImage, err := docker.NewRemoteImage(e.Ctx(), fmt.Sprintf("%v-image", args.Name), &docker.RemoteImageArgs{
		Name: pulumi.String("geerlingguy/docker-ubuntu2204-ansible:latest"),
	}, utils.MergeOptions(opts, e.WithProviders(config.ProviderDocker))...)
	if err != nil {
		return nil, err
	}

	// Create a Docker network
	network, err := docker.NewNetwork(e.Ctx(), "network", &docker.NetworkArgs{
		Name: pulumi.String(fmt.Sprintf("local-e2e-%v", e.Ctx().Stack())),
	}, utils.MergeOptions(opts, e.WithProviders(config.ProviderDocker))...)
	if err != nil {
		return nil, err
	}

	// Create Agent container
	_, err = docker.NewContainer(e.Ctx(), "agent-container", &docker.ContainerArgs{
		Name:         pulumi.String(fmt.Sprintf("agent-%v", e.Ctx().Stack())),
		Image:        hostImage.RepoDigest,
		CgroupnsMode: pulumi.String("host"),
		Privileged:   pulumi.Bool(true),
		Mounts: docker.ContainerMountArray{
			&docker.ContainerMountArgs{
				Target:   pulumi.String("/sys/fs/cgroup"),
				Source:   pulumi.String("/sys/fs/cgroup"),
				ReadOnly: pulumi.Bool(false),
				Type:     pulumi.String("bind"),
			},
		},
		Rm:          pulumi.Bool(true),
		StopTimeout: pulumi.IntPtr(5),
		NetworksAdvanced: &docker.ContainerNetworksAdvancedArray{
			&docker.ContainerNetworksAdvancedArgs{
				Name: network.Name,
				Aliases: pulumi.StringArray{
					pulumi.String("agent"),
				},
			},
		},
	}, utils.MergeOptions(opts, e.WithProviders(config.ProviderDocker))...)
	if err != nil {
		return nil, err
	}
	return nil, nil
}
