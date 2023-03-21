package vm

import (
	"github.com/DataDog/test-infra-definitions/command"
	"github.com/DataDog/test-infra-definitions/common/config"
	commonos "github.com/DataDog/test-infra-definitions/common/os"
	"github.com/DataDog/test-infra-definitions/common/utils"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type VM interface {
	GetRunner() *command.Runner
	GetCommonEnvironment() *config.CommonEnvironment
	GetOS() commonos.OS
}

type genericVM struct {
	runner *command.Runner
	env    *config.CommonEnvironment
	os     commonos.OS
}

func NewGenericVM(
	name string,
	env config.Environment,
	instanceIP pulumi.StringInput,
	os commonos.OS,
) (VM, error) {
	commonEnv := env.GetCommonEnvironment()
	ctx := commonEnv.Ctx

	readyFunc := func(r *command.Runner) (*remote.Command, error) { return command.WaitForCloudInit(ctx, r) }
	if os.GetType() == commonos.WindowsType {
		cmd := `for ($i = 0; $i -le 120; $i++) { 
			$service = Get-Service -Name sshd -ErrorAction SilentlyContinue;
			if ($service -ne $null -and $service.Status -eq "Running") {
				exit 0
			}
			Start-Sleep -Second 1
		}		
		exit 1
		`
		// Most of the waiting time is waiting the remote command to run.
		// This command is still relevant to make sure ssh is running.
		readyFunc = func(r *command.Runner) (*remote.Command, error) {
			return r.Command(
				"wait-openssh-require-win2019-win10-or-above",
				&command.Args{
					Create: pulumi.String(cmd),
				})
		}
	}
	connection, runner, err := createRunner(ctx, env, instanceIP, os.GetSSHUser(), readyFunc)
	if err != nil {
		return nil, err
	}

	ctx.Export("connection", connection)

	return &genericVM{
		runner: runner,
		env:    commonEnv,
		os:     os,
	}, nil
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
