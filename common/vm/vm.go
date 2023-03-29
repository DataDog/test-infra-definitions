package vm

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/command"
	"github.com/DataDog/test-infra-definitions/common/config"
	commonos "github.com/DataDog/test-infra-definitions/common/os"
	"github.com/DataDog/test-infra-definitions/common/utils"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type VM interface {
	GetRunner() *command.Runner
	GetCommonEnvironment() *config.CommonEnvironment
	GetOS() commonos.OS
	GetClientDataDeserializer() func(auto.UpResult) (*ClientData, error)
}

type genericVM struct {
	runner   *command.Runner
	env      *config.CommonEnvironment
	os       commonos.OS
	stackKey string
}

func NewGenericVM(
	name string,
	env config.Environment,
	instanceIP pulumi.StringInput,
	os commonos.OS,
) (VM, error) {
	commonEnv := env.GetCommonEnvironment()
	ctx := commonEnv.Ctx

	readyFunc := func(r *command.Runner) (*remote.Command, error) { return command.WaitForCloudInit(r) }
	if os.GetType() == commonos.WindowsType {
		readyFunc = func(r *command.Runner) (*remote.Command, error) {
			// Wait until a command can be executed.
			return r.Command(
				"wait-openssh-require-win2019-win10-or-above",
				&command.Args{
					Create: pulumi.String(`Write-Host "Ready"`),
				})
		}
	}
	connection, runner, err := createRunner(env, instanceIP, os.GetSSHUser(), readyFunc)
	if err != nil {
		return nil, err
	}

	stackKey := fmt.Sprintf("%v-connection", name)
	ctx.Export(stackKey, connection)

	return &genericVM{
		runner:   runner,
		env:      commonEnv,
		os:       os,
		stackKey: stackKey,
	}, nil
}

type ClientData struct {
	Connection utils.Connection
}

func (vm *genericVM) GetClientDataDeserializer() func(auto.UpResult) (*ClientData, error) {
	return func(result auto.UpResult) (*ClientData, error) {
		connection, err := utils.NewConnection(result, vm.stackKey)
		return &ClientData{Connection: connection}, err
	}
}

func (vm *genericVM) GetRunner() *command.Runner {
	return vm.runner
}

func (vm *genericVM) GetCommonEnvironment() *config.CommonEnvironment {
	return vm.env
}

func (vm *genericVM) GetOS() commonos.OS {
	return vm.os
}

func createRunner(
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
