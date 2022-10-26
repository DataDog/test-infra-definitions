package command

import (
	"fmt"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type PackageManager interface {
	Ensure(packageRef string, updateDB bool)
}

type AptManager struct {
	ctx           *pulumi.Context
	connection    remote.ConnectionInput
	connName      string
	updateCommand *remote.Command
	environment   pulumi.StringMapInput
}

func NewAptManager(ctx *pulumi.Context, name string, connection remote.ConnectionInput) *AptManager {
	return &AptManager{
		ctx:        ctx,
		connection: connection,
		connName:   name,
	}
}

func (m *AptManager) Ensure(packageRef string) (*remote.Command, error) {
	updateDB, err := m.updateDB()
	if err != nil {
		return nil, err
	}
	create := fmt.Sprintf("apt install -y %s", packageRef)
	return m.command(create, "", "", pulumi.DependsOn([]pulumi.Resource{updateDB}))
}

func (m *AptManager) updateDB() (*remote.Command, error) {
	var err error
	if m.updateCommand == nil {
		m.updateCommand, err = m.command("apt update", "", "")
	}
	return m.updateCommand, err
}

func (m *AptManager) command(createCmd, updateCmd, deleteCmd string, opts ...pulumi.ResourceOption) (*remote.Command, error) {
	if len(createCmd) > 0 {
		createCmd = "DEBIAN_FRONTEND=noninteractive " + createCmd
	}
	if len(updateCmd) > 0 {
		updateCmd = "DEBIAN_FRONTEND=noninteractive " + updateCmd
	}
	if len(deleteCmd) > 0 {
		deleteCmd = "DEBIAN_FRONTEND=noninteractive " + deleteCmd
	}

	return remote.NewCommand(m.ctx,
		UniqueCommandName(m.connName, createCmd, updateCmd, deleteCmd),
		&remote.CommandArgs{
			Connection:  m.connection,
			Create:      pulumi.String(createCmd),
			Update:      pulumi.String(updateCmd),
			Delete:      pulumi.String(deleteCmd),
			Environment: m.environment,
		}, opts...)
}
