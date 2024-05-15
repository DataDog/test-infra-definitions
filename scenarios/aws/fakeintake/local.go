package fakeintake

import (
	"fmt"
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components"
	"github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
	"github.com/DataDog/test-infra-definitions/resources/local/docker"

	pdocker "github.com/pulumi/pulumi-docker/sdk/v4/go/docker"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func NewLocalInstance(e docker.Environment, name string, option ...Option) (*fakeintake.Fakeintake, error) {
	params, paramsErr := NewParams(option...)
	if paramsErr != nil {
		return nil, paramsErr
	}

	return components.NewComponent(&e, e.Namer.ResourceName(name), func(fi *fakeintake.Fakeintake) error {
		opts := []pulumi.ResourceOption{pulumi.Parent(fi)}

		// Get fake intake image
		fiImage, err := pdocker.NewRemoteImage(e.Ctx(), fmt.Sprintf("%v-fakeintake-image", name), &pdocker.RemoteImageArgs{
			Name:        pulumi.String(params.ImageURL),
			KeepLocally: pulumi.Bool(true),
			Platform:    pulumi.String("linux/arm64"),
		}, utils.MergeOptions(opts, e.WithProviders(config.ProviderDocker))...)
		if err != nil {
			return err
		}

		// Start fake intake is a container
		fiContainer, err := pdocker.NewContainer(e.Ctx(), "fakeintakeContainer", &pdocker.ContainerArgs{
			Name:        pulumi.String(fmt.Sprintf("fakeintake-%v", e.Ctx().Stack())),
			Image:       fiImage.ImageId,
			Rm:          pulumi.Bool(false),
			StopTimeout: pulumi.IntPtr(5),
			Hostname:    pulumi.String("fakeintake"),
			NetworksAdvanced: &pdocker.ContainerNetworksAdvancedArray{
				&pdocker.ContainerNetworksAdvancedArgs{
					Name: e.DockerNetwork.Name,
				},
			},
		}, utils.MergeOptions(opts, e.WithProviders(config.ProviderDocker))...)
		if err != nil {
			return err
		}

		fi.Scheme = "http"
		fi.Port = 80
		fi.Host = fiContainer.Hostname
		fi.URL = fi.Host.ApplyT(func(v string) pulumi.StringOutput {
			return pulumi.Sprintf("%s://%s", fi.Scheme, v)
		}).(pulumi.StringOutput)

		return nil
	})
}
