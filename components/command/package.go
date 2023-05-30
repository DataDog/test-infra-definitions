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

type genericPackageManager struct {
	namer           namer.Namer
	updateDBCommand *remote.Command
	runner          *Runner
	opts            []pulumi.ResourceOption
	installCmd      string
	updateCmd       string
	env             pulumi.StringMap
}

func NewGenericPackageManager(
	runner *Runner,
	name string,
	installCmd string,
	updateCmd string,
	env pulumi.StringMap) PackageManager {
	packageManager := &genericPackageManager{
		namer:      namer.NewNamer(runner.e.Ctx, name),
		runner:     runner,
		installCmd: installCmd,
		updateCmd:  updateCmd,
		env:        env,
	}

	return packageManager
}

func (m *genericPackageManager) Ensure(packageRef string, opts ...pulumi.ResourceOption) (*remote.Command, error) {
	opts = append(opts, m.opts...)
	if m.updateCmd != "" {
		updateDB, err := m.updateDB(opts)
		if err != nil {
			return nil, err
		}

		opts = append(opts, utils.PulumiDependsOn(updateDB))
	}
	installCmd := fmt.Sprintf("%s %s", m.installCmd, packageRef)
	cmd, err := m.runner.Command(
		m.namer.ResourceName("install-"+packageRef, utils.StrHash(installCmd)),
		&Args{
			Create:      pulumi.String(installCmd),
			Environment: m.env,
			Sudo:        true,
		},
		opts...)
	if err != nil {
		return nil, err
	}
	// Make sure the package manager isn't running in parallel
	m.opts = append(m.opts, utils.PulumiDependsOn(cmd))
	return cmd, nil
}

func (m *genericPackageManager) updateDB(opts []pulumi.ResourceOption) (*remote.Command, error) {
	if m.updateDBCommand != nil {
		return m.updateDBCommand, nil
	}

	c, err := m.runner.Command(
		m.namer.ResourceName("update"),
		&Args{
			Create:      pulumi.String(m.updateCmd),
			Environment: m.env,
			Sudo:        true,
		}, opts...)
	if err == nil {
		m.updateDBCommand = c
	}

	return c, err
}
