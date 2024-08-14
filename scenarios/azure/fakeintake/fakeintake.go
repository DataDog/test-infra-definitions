package fakeintake

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
	"github.com/DataDog/test-infra-definitions/components/docker"
	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/resources/azure"
	"github.com/DataDog/test-infra-definitions/scenarios/azure/compute"
	app "github.com/pulumi/pulumi-azure-native-sdk/app/v2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func NewContainerAppInstance(e *azure.Environment) (*fakeintake.Fakeintake, error) {
	return components.NewComponent(e, "fakeintake", func(fi *fakeintake.Fakeintake) error {
		// environment, err := app.NewManagedEnvironment(e.Ctx(), "fakeintake-2", &app.ManagedEnvironmentArgs{
		// 	ResourceGroupName: pulumi.String(e.DefaultResourceGroup()),
		// 	VnetConfiguration: &app.VnetConfigurationArgs{
		// 		Internal:               pulumi.Bool(true),
		// 		InfrastructureSubnetId: pulumi.String(e.DefaultSubnet()),
		// 	},
		// }, e.WithProviders(config.ProviderAzure))

		// if err != nil {
		// 	return err
		// }

		containerApp, err := app.NewContainerApp(e.Ctx(), "fakeintake-2", &app.ContainerAppArgs{
			ResourceGroupName: pulumi.String(e.DefaultResourceGroup()),
			EnvironmentId:     pulumi.String("/subscriptions/9972cab2-9e99-419b-a683-86bfa77b3df1/resourceGroups/dd-agent-sandbox/providers/Microsoft.App/managedEnvironments/fakeintake-274b947f2"),
			Template: &app.TemplateArgs{
				Containers: app.ContainerArray{
					&app.ContainerArgs{
						Name:  pulumi.String("fakeintake"),
						Image: pulumi.String("public.ecr.aws/datadog/fakeintake:latest"),
					},
				},
			},
		}, e.WithProviders(config.ProviderAzure))

		if err != nil {
			return err
		}

		fi.Host = containerApp.LatestRevisionFqdn
		fi.Port = 443
		fi.Scheme = "https"
		fi.URL = pulumi.Sprintf("https://%s", containerApp.LatestRevisionFqdn)

		return nil
	})
}

func NewVMInstance(e azure.Environment) (*fakeintake.Fakeintake, error) {
	return components.NewComponent(&e, "fakeintake", func(fi *fakeintake.Fakeintake) error {

		vm, err := compute.NewVM(e, "fakeintake", compute.WithOS(os.UbuntuDefault), compute.WithPulumiResourceOptions(pulumi.Parent(fi)))
		if err != nil {
			return err
		}
		manager, err := docker.NewManager(&e, vm, pulumi.Parent(vm))
		if err != nil {
			return err
		}

		_, err = vm.OS.Runner().Command("docker_run_fakeintake", &command.Args{
			Create: pulumi.String("docker run --restart unless-stopped --name fakeintake -d -p 80:80 public.ecr.aws/datadog/fakeintake:latest"),
			Delete: pulumi.String("docker stop fakeintake"),
		}, utils.PulumiDependsOn(manager), pulumi.DeleteBeforeReplace(true))
		if err != nil {
			return err
		}

		fi.Host = vm.Address
		fi.Scheme = "http"
		fi.Port = 80
		fi.URL = pulumi.Sprintf("%s://%s:%v", fi.Scheme, vm.Address, fi.Port)

		return nil
	})
}
