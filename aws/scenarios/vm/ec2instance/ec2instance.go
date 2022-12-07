package ec2instance

import (
	"github.com/DataDog/test-infra-definitions/aws"
	awsEc2 "github.com/DataDog/test-infra-definitions/aws/ec2/ec2"
	"github.com/DataDog/test-infra-definitions/aws/scenarios/vm/agentinstall"
	"github.com/DataDog/test-infra-definitions/aws/scenarios/vm/os"
	"github.com/DataDog/test-infra-definitions/command"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Ec2Instance struct {
	runner *command.Runner
}

// NewEc2Instance creates a new EC2 instance. By default use WithOS(os.UbuntuOS, os.AMD64Arch).
func NewEc2Instance(ctx *pulumi.Context, options ...func(*Params) error) (*Ec2Instance, error) {
	e, err := aws.AWSEnvironment(ctx)
	if err != nil {
		return nil, err
	}

	params, err := newParams(e, options...)
	if err != nil {
		return nil, err
	}

	instance, err := awsEc2.NewEC2Instance(
		e,
		e.CommonNamer.ResourceName("ec2-instance"),
		params.ami,
		params.os.GetAMIArch(params.arch),
		params.instanceType,
		params.keyPair,
		params.userData,
		params.os.GetTenancy())

	if err != nil {
		return nil, err
	}

	connection, runner, err := createRunner(ctx, e, instance, params.os)
	if err != nil {
		return nil, err
	}

	if params.optionalAgentInstallParams != nil {
		agentinstall.Install(runner, e, params.optionalAgentInstallParams, params.os)
	}
	e.Ctx.Export("instance-ip", instance.PrivateIp)
	e.Ctx.Export("connection", connection)

	return &Ec2Instance{runner: runner}, nil
}

func createRunner(ctx *pulumi.Context, env aws.Environment, instance *ec2.Instance, os os.OS) (remote.ConnectionOutput, *command.Runner, error) {
	connection, err := createConnection(instance, os.GetSSHUser(), env)
	if err != nil {
		return remote.ConnectionOutput{}, nil, err
	}

	runner, err := command.NewRunner(*env.CommonEnvironment, ctx.Stack()+"-conn", connection, func(r *command.Runner) (*remote.Command, error) {
		return command.WaitForCloudInit(ctx, r)
	})
	if err != nil {
		return remote.ConnectionOutput{}, nil, err
	}
	return connection, runner, nil
}

func createConnection(instance *ec2.Instance, user string, e aws.Environment) (remote.ConnectionOutput, error) {
	connection := remote.ConnectionArgs{
		Host: instance.PrivateIp,
	}

	if err := utils.ConfigureRemoteSSH(user, e.DefaultPrivateKeyPath(), e.DefaultPrivateKeyPassword(), "", &connection); err != nil {
		return remote.ConnectionOutput{}, err
	}

	return connection.ToConnectionOutput(), nil
}
