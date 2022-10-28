package command

import (
	"fmt"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type PackageManager interface {
	Ensure(packageRef string, opts ...pulumi.ResourceOption) (*remote.Command, error)
}

type AptManager struct {
	ctx             *pulumi.Context
	updateDBCommand *remote.Command
	runner          *Runner
}

func NewAptManager(ctx *pulumi.Context, runner *Runner) *AptManager {
	apt := &AptManager{
		ctx:    ctx,
		runner: runner,
	}

	if apt.runner.defaultEnv == nil {
		apt.runner.defaultEnv = pulumi.StringMap{}
	}
	apt.runner.defaultEnv["DEBIAN_FRONTEND"] = pulumi.String("noninteractive")

	return apt
}

func (m *AptManager) Ensure(packageRef string, opts ...pulumi.ResourceOption) (*remote.Command, error) {
	updateDB, err := m.updateDB()
	if err != nil {
		return nil, err
	}

	opts = append(opts, pulumi.DependsOn([]pulumi.Resource{updateDB}))
	installCmd := fmt.Sprintf("apt install -y %s", packageRef)
	return m.runner.Command(m.ctx, UniqueCommandName("apt-install", installCmd, "", ""), pulumi.String(installCmd), nil, nil, nil, true, opts...)
}

func (m *AptManager) updateDB() (*remote.Command, error) {
	if m.updateDBCommand != nil {
		return m.updateDBCommand, nil
	}

	c, err := m.runner.Command(m.ctx, "updatedb", pulumi.String("apt update"), nil, nil, nil, true)
	if err == nil {
		m.updateDBCommand = c
	}

	return c, err
}
