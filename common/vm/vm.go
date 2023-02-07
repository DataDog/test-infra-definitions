package vm

import (
	"github.com/DataDog/test-infra-definitions/command"
	"github.com/DataDog/test-infra-definitions/common/config"
	commonos "github.com/DataDog/test-infra-definitions/common/os"
	"github.com/DataDog/test-infra-definitions/common/utils"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func NewVM(
	name string,
	env config.Environment,
	instanceIP pulumi.StringInput,
	os commonos.OS,
) (VM, error) {
	commonEnv := env.GetCommonEnvironment()
	ctx := commonEnv.Ctx

	readyFunc := func(r *command.Runner) (*remote.Command, error) { return command.WaitForCloudInit(ctx, r) }
	if os.GetType() == commonos.WindowsType {
		// On Windows, there is no equivalent of cloud init, but the code wait until ssh connection is ready
		// so it is OK to not have a ready function.
		readyFunc = nil
	}
	connection, runner, err := createRunner(ctx, env, instanceIP, os.GetSSHUser(), readyFunc)
	if err != nil {
		return nil, err
	}

	ctx.Export("connection", connection)

	rawVM := rawVM{
		runner: runner,
		env:    commonEnv,
		os:     os,
	}

	switch os.GetType() {
	case commonos.UbuntuType:
		return &UbuntuVM{
			rawVM:      rawVM,
			aptManager: command.NewAptManager(runner),
		}, nil
	case commonos.WindowsType:
		return &rawVM, nil
	case commonos.OtherType:
		return &rawVM, nil
	default:
		return &rawVM, nil
	}
}

func createRunner(
	ctx *pulumi.Context,
	env config.Environment,
	instanceIP pulumi.StringInput,
	sshUser string,
	readyFunc func(*command.Runner) (*remote.Command, error),
) (remote.ConnectionOutput, *command.Runner, error) {
	connection, err := createConnection(instanceIP, sshUser, env)
	if err != nil {
		return remote.ConnectionOutput{}, nil, err
	}

	commonEnv := env.GetCommonEnvironment()
	runner, err := command.NewRunner(
		*commonEnv,
		commonEnv.CommonNamer.ResourceName("connection"),
		connection,
		readyFunc)
	if err != nil {
		return remote.ConnectionOutput{}, nil, err
	}
	return connection, runner, nil
}

func createConnection(ip pulumi.StringInput, user string, env config.Environment) (remote.ConnectionOutput, error) {
	connection := remote.ConnectionArgs{
		Host: ip,
	}

	if err := utils.ConfigureRemoteSSH(user, env.DefaultPrivateKeyPath(), env.DefaultPrivateKeyPassword(), "", &connection); err != nil {
		return remote.ConnectionOutput{}, err
	}

	return connection.ToConnectionOutput(), nil
}

type VM interface {
	GetRunner() *command.Runner
	GetCommonEnvironment() *config.CommonEnvironment
	GetOS() commonos.OS
}

type rawVM struct {
	runner *command.Runner
	env    *config.CommonEnvironment
	os     commonos.OS
}

func (vm *rawVM) GetRunner() *command.Runner {
	return vm.runner
}

func (vm *rawVM) GetCommonEnvironment() *config.CommonEnvironment {
	return vm.env
}

func (vm *rawVM) GetOS() commonos.OS {
	return vm.os
}

type UbuntuVM struct {
	aptManager *command.AptManager
	rawVM
}

func (vm *UbuntuVM) GetAptManager() *command.AptManager {
	return vm.aptManager
}
