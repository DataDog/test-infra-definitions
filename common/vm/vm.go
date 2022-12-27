package vm

import (
	"github.com/DataDog/test-infra-definitions/command"
	"github.com/DataDog/test-infra-definitions/common/agentinstall"
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/os"
	"github.com/DataDog/test-infra-definitions/common/utils"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func InitVM(
	ctx *pulumi.Context,
	env config.Environment,
	instanceIP pulumi.StringInput,
	os os.OS,
	optionalAgentInstallParams *agentinstall.Params,
) (*command.Runner, error) {
	connection, runner, err := createRunner(ctx, env, instanceIP, os.GetSSHUser())
	if err != nil {
		return nil, err
	}

	if optionalAgentInstallParams != nil {
		agentinstall.Install(runner, env.GetCommonEnvironment(), optionalAgentInstallParams, os)
	}
	ctx.Export("connection", connection)

	return runner, nil
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
		commonEnv.CommonNamer.ResourceName(ctx.Stack()+"-conn"),
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
