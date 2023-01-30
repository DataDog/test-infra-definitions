package command

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/common/namer"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type PackageManager interface {
	Ensure(packageRef string, opts ...pulumi.ResourceOption) (*remote.Command, error)
}

type AptManager struct {
	namer           namer.Namer
	updateDBCommand *remote.Command
	runner          *Runner
	env             pulumi.StringMap
	opts            []pulumi.ResourceOption
}

func NewAptManager(runner *Runner, opts ...pulumi.ResourceOption) *AptManager {
	apt := &AptManager{
		namer:  namer.NewNamer(runner.e.Ctx, "apt"),
		runner: runner,
		env: pulumi.StringMap{
			"DEBIAN_FRONTEND": pulumi.String("noninteractive"),
		},
		opts: opts,
	}

	return apt
}

func (m *AptManager) Ensure(packageRef string, opts ...pulumi.ResourceOption) (*remote.Command, error) {
	opts = append(opts, m.opts...)
	updateDB, err := m.updateDB(opts)
	if err != nil {
		return nil, err
	}

	opts = append(opts, utils.PulumiDependsOn(updateDB))
	installCmd := fmt.Sprintf("apt-get install -y %s", packageRef)
	cmd, err := m.runner.Command(
		m.namer.ResourceName("install", utils.StrHash(installCmd)),
		&Args{
			Create:      pulumi.String(installCmd),
			Environment: m.env,
			Sudo:        true,
		},
		opts...)
	if err != nil {
		return nil, err
	}
	// Make sure apt-get install doesn't run in parallel
	m.opts = append(m.opts, utils.PulumiDependsOn(cmd))
	return cmd, nil
}

func (m *AptManager) updateDB(opts []pulumi.ResourceOption) (*remote.Command, error) {
	if m.updateDBCommand != nil {
		return m.updateDBCommand, nil
	}

	c, err := m.runner.Command(
		m.namer.ResourceName("update"),
		&Args{
			Create:      pulumi.String("apt-get update -y"),
			Sudo:        true,
			Environment: m.env,
		}, opts...)
	if err == nil {
		m.updateDBCommand = c
	}

	return c, err
}
