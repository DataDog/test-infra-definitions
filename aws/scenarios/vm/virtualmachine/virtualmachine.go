package virtualmachine

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/aws"
	awsEc2 "github.com/DataDog/test-infra-definitions/aws/ec2/ec2"
	"github.com/DataDog/test-infra-definitions/command"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type VirtualMachine struct {
	runner *command.Runner
}

func NewVirtualMachine(ctx *pulumi.Context, options ...func(*Params) error) (*VirtualMachine, error) {
	e, err := aws.AWSEnvironment(ctx)
	if err != nil {
		return nil, err
	}

	params, err := createParams(e, options...)
	if err != nil {
		return nil, err
	}
	instance, err := awsEc2.NewEC2Instance(e, ctx.Stack(), params.ami, string(params.arch), params.instanceType, params.keyPair, params.userData)

	if err != nil {
		return nil, err
	}

	connection, runner, err := createRunner(ctx, e, instance, params.os)
	if err != nil {
		return nil, err
	}

	e.Ctx.Export("instance-ip", instance.PrivateIp)
	e.Ctx.Export("connection", connection)

	return &VirtualMachine{runner: runner}, nil
}

func createParams(env aws.Environment, options ...func(*Params) error) (*Params, error) {
	params := &Params{
		instanceType: "t3.large",
		keyPair:      "agent-ci-sandbox",
		env:          env,
	}
	options = append([]func(*Params) error{WithOS(LinuxOS, AMD64Arch)}, options...)
	for _, o := range options {
		if err := o(params); err != nil {
			return nil, err
		}
	}
	return params, nil
}

func createRunner(ctx *pulumi.Context, env aws.Environment, instance *ec2.Instance, os OS) (remote.ConnectionOutput, *command.Runner, error) {
	sshUser := ""
	switch os {
	case LinuxOS:
		sshUser = "ubuntu"
	case MacOS:
		sshUser = "ec2-user"
	case WindowsOS:
		return remote.ConnectionOutput{}, nil, fmt.Errorf("%v is not yet supported", os)
	}

	connection, err := createConnection(instance, sshUser, env)
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
