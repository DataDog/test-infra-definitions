package command

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/common/namer"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type GenericPackageManager struct {
	namer           namer.Namer
	updateDBCommand Command
	runner          Runner
	opts            []pulumi.ResourceOption
	installCmd      string
	updateCmd       string
	env             pulumi.StringMap
}

func NewGenericPackageManager(
	runner Runner,
	name string,
	installCmd string,
	updateCmd string,
	env pulumi.StringMap,
) *GenericPackageManager {
	packageManager := &GenericPackageManager{
		namer:      namer.NewNamer(runner.Environment().Ctx(), name),
		runner:     runner,
		installCmd: installCmd,
		updateCmd:  updateCmd,
		env:        env,
	}

	return packageManager
}

func (m *GenericPackageManager) Ensure(packageRef string, transform Transformer, checkBinary string, opts ...pulumi.ResourceOption) (Command, error) {
	opts = append(opts, m.opts...)
	if m.updateCmd != "" {
		updateDB, err := m.updateDB(opts)
		if err != nil {
			return nil, err
		}

		opts = append(opts, utils.PulumiDependsOn(updateDB))
	}
	var cmdStr string
	if checkBinary != "" {
		cmdStr = fmt.Sprintf("bash -c 'command -v %s || %s %s'", checkBinary, m.installCmd, packageRef)
	} else {
		cmdStr = fmt.Sprintf("%s %s", m.installCmd, packageRef)
	}

	cmdName := m.namer.ResourceName("install-"+packageRef, utils.StrHash(cmdStr))
	var cmdArgs RunnerCommandArgs = &Args{
		Create:      pulumi.String(cmdStr),
		Environment: m.env,
		Sudo:        true,
	}

	// If a transform is provided, use it to modify the command name and args
	if transform != nil {
		cmdName, cmdArgs = transform(cmdName, cmdArgs)
	}

	cmd, err := m.runner.Command(cmdName, cmdArgs, opts...)
	if err != nil {
		return nil, err
	}

	// Make sure the package manager isn't running in parallel
	m.opts = append(m.opts, utils.PulumiDependsOn(cmd))
	return cmd, nil
}

func (m *GenericPackageManager) updateDB(opts []pulumi.ResourceOption) (Command, error) {
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
