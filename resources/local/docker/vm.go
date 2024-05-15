package docker

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/pulumi/pulumi-docker/sdk/v4/go/docker"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type VMArgs struct {
	Name string
	// Attributes you need when you will actually create the VM
}

func NewInstance(e Environment, args VMArgs, opts ...pulumi.ResourceOption) (*docker.Container, error) {
	//hostImage, err := docker.NewRemoteImage(e.Ctx(), fmt.Sprintf("%v-image", args.Name), &docker.RemoteImageArgs{
	//	Name: pulumi.String("fake-host"),
	//}, utils.MergeOptions(opts, e.WithProviders(config.ProviderDocker))...)
	//if err != nil {
	//	return nil, err
	//}

	// TODO: Fix as current hack due to fact not published images yet
	hostImage, err := docker.NewImage(e.Ctx(), fmt.Sprintf("%v-image", args.Name), &docker.ImageArgs{
		Build: &docker.DockerBuildArgs{
			Context:    pulumi.String("/data/dev/DataDog/test-infra-definitions/scenarios/local/docker/containers"),
			Dockerfile: pulumi.String("/data/dev/DataDog/test-infra-definitions/scenarios/local/docker/containers/Dockerfile"),
			Platform:   pulumi.String("linux/arm64"),
		},
		SkipPush:  pulumi.Bool(true),
		ImageName: pulumi.String("fake-host"),
	}, utils.MergeOptions(opts, e.WithProviders(config.ProviderDocker))...)
	if err != nil {
		return nil, err
	}

	// Create Agent container
	instance, err := docker.NewContainer(e.Ctx(), "agent-container", &docker.ContainerArgs{
		Name:         pulumi.String(fmt.Sprintf("agent-%v", e.Ctx().Stack())),
		Image:        hostImage.ImageName,
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
		Rm:          pulumi.Bool(false),
		StopTimeout: pulumi.IntPtr(5),
		Ports: docker.ContainerPortArray{
			&docker.ContainerPortArgs{
				Internal: pulumi.Int(22),
				Protocol: pulumi.String("tcp"),
			},
		},
		NetworksAdvanced: &docker.ContainerNetworksAdvancedArray{
			&docker.ContainerNetworksAdvancedArgs{
				Name: e.DockerNetwork.Name,
				Aliases: pulumi.StringArray{
					pulumi.String("agent"),
				},
			},
		},
	}, utils.MergeOptions(opts, e.WithProviders(config.ProviderDocker))...)
	if err != nil {
		return nil, err
	}
	return instance, nil
}
