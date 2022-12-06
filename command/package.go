package command

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/common"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type PackageManager interface {
	Ensure(packageRef string, opts ...pulumi.ResourceOption) (*remote.Command, error)
}

type AptManager struct {
	namer           common.Namer
	updateDBCommand *remote.Command
	runner          *Runner
	env             pulumi.StringMap
}

func NewAptManager(runner *Runner) *AptManager {
	apt := &AptManager{
		namer:  common.NewNamer(runner.e.Ctx, "apt"),
		runner: runner,
		env: pulumi.StringMap{
			"DEBIAN_FRONTEND": pulumi.String("noninteractive"),
		},
	}

	return apt
}

func (m *AptManager) Ensure(packageRef string, opts ...pulumi.ResourceOption) (*remote.Command, error) {
	updateDB, err := m.updateDB()
	if err != nil {
		return nil, err
	}

	opts = append(opts, pulumi.DependsOn([]pulumi.Resource{updateDB}))
	installCmd := fmt.Sprintf("apt-get install -y %s", packageRef)
	return m.runner.Command(
		m.namer.ResourceName("install", utils.StrHash(installCmd)),
		&CommandArgs{
			Create:      pulumi.String(installCmd),
			Environment: m.env,
			Sudo:        true,
		},
		opts...)
}

func (m *AptManager) updateDB() (*remote.Command, error) {
	if m.updateDBCommand != nil {
		return m.updateDBCommand, nil
	}

	c, err := m.runner.Command(
		m.namer.ResourceName("update"),
		&CommandArgs{
			Create:      pulumi.String("apt-get update -y"),
			Sudo:        true,
			Environment: m.env,
		})
	if err == nil {
		m.updateDBCommand = c
	}

	return c, err
}
