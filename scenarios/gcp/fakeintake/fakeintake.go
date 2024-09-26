package fakeintake

import (
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
	"github.com/DataDog/test-infra-definitions/components/docker"
	"github.com/DataDog/test-infra-definitions/resources/gcp"
	"github.com/DataDog/test-infra-definitions/scenarios/gcp/compute"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func NewVMInstance(e gcp.Environment, option ...Option) (*fakeintake.Fakeintake, error) {
	params, paramsErr := NewParams(option...)
	if paramsErr != nil {
		return nil, paramsErr
	}

	return components.NewComponent(&e, "fakeintake", func(fi *fakeintake.Fakeintake) error {

		vm, err := compute.NewVM(e, "fakeintake")
		if err != nil {
			return err
		}
		manager, err := docker.NewManager(&e, vm, pulumi.Parent(vm))
		if err != nil {
			return err
		}
		cmdArgs := []string{}

		if params.DDDevForwarding {
			cmdArgs = append(cmdArgs, "--dddev-forward")
		}

		_, err = vm.OS.Runner().Command("docker_run_fakeintake", &command.Args{
			Create: pulumi.Sprintf("docker run --restart unless-stopped --name fakeintake -d -p 80:80 -e DD_API_KEY=%s %s %s", e.AgentAPIKey(), params.ImageURL, cmdArgs),
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
