package fakeintake

import (
	"strings"

	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
	"github.com/DataDog/test-infra-definitions/components/docker"
	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/resources/azure"
	"github.com/DataDog/test-infra-definitions/scenarios/azure/compute"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func NewVMInstance(e azure.Environment, opts ...Option) (*fakeintake.Fakeintake, error) {
	params, err := NewParams(opts...)
	if err != nil {
		return nil, err
	}

	return components.NewComponent(&e, "fakeintake", func(fi *fakeintake.Fakeintake) error {

		vm, err := compute.NewVM(e, "fakeintake", compute.WithOS(os.UbuntuDefault), compute.WithPulumiResourceOptions(pulumi.Parent(fi)))
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
			Create: pulumi.Sprintf("docker run --restart unless-stopped --name fakeintake -d -p 80:80 -e DD_API_KEY='%s' %s %s", e.AgentAPIKey(), params.ImageURL, strings.Join(cmdArgs, " ")),
			Delete: pulumi.String("docker stop fakeintake && docker rm fakeintake"),
		}, utils.PulumiDependsOn(manager), pulumi.DeleteBeforeReplace(true), pulumi.ReplaceOnChanges([]string{"*"}))
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
