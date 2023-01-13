package vm

import (
	"github.com/DataDog/test-infra-definitions/command"
	"github.com/DataDog/test-infra-definitions/common/agentinstall"
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
	optionalAgentInstallParams *agentinstall.Params,
) (VM, error) {
	commonEnv := env.GetCommonEnvironment()
	ctx := commonEnv.Ctx
	connection, runner, err := createRunner(ctx, env, instanceIP, os.GetSSHUser())
	if err != nil {
		return nil, err
	}

	if optionalAgentInstallParams != nil {
		agentinstall.Install(runner, commonEnv, optionalAgentInstallParams, os)
	}
	ctx.Export("connection", connection)

	rawVM := rawVM{
		runner: runner,
		env:    commonEnv,
	}
	switch os.GetOSType() {
	case commonos.UbuntuOS:
		return &UbuntuVM{
			rawVM:      rawVM,
			aptManager: *command.NewAptManager(runner),
		}, nil
	default:
		return &rawVM, nil
	}
}

func createRunner(
	ctx *pulumi.Context,
	env config.Environment,
	instanceIP pulumi.StringInput,
	sshUser string,
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
		func(r *command.Runner) (*remote.Command, error) {
			return command.WaitForCloudInit(ctx, r)
		})
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
}

type rawVM struct {
	runner *command.Runner
	env    *config.CommonEnvironment
}

func (vm *rawVM) GetRunner() *command.Runner {
	return vm.runner
}

func (vm *rawVM) GetCommonEnvironment() *config.CommonEnvironment {
	return vm.env
}

type UbuntuVM struct {
	aptManager command.AptManager
	rawVM
}

func (vm *UbuntuVM) GetAptManager() *command.AptManager {
	return &vm.aptManager
}
