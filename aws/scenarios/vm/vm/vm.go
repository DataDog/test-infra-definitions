package vm

import (
	"github.com/DataDog/test-infra-definitions/aws"
	awsEc2 "github.com/DataDog/test-infra-definitions/aws/ec2/ec2"
	"github.com/DataDog/test-infra-definitions/command"
	"github.com/DataDog/test-infra-definitions/common/config"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type VM struct {
	Context     *pulumi.Context
	Runner      *command.Runner
	Environment *config.CommonEnvironment
	// TODO add file manager as soon as https://github.com/DataDog/test-infra-definitions/pull/9 is merged
	// FileManager   *command.FileManager
	DockerManager *command.DockerManager
}

/*
NewVM Creates an instance of VM on aws
Returns a VM object with most operation mamagers to install docker containers
and run generic commands.
*/
func NewVM(ctx *pulumi.Context) (*VM, error) {
	e, err := aws.AWSEnvironment(ctx)
	if err != nil {
		return nil, err
	}

	instance, conn, err := awsEc2.NewDefaultEC2Instance(e, ctx.Stack(), e.DefaultInstanceType())
	if err != nil {
		return nil, err
	}

	runner, err := command.NewRunner(ctx.Stack()+"-conn", conn, func(r *command.Runner) (*remote.Command, error) {
		return command.WaitForCloudInit(ctx, r)
	})
	if err != nil {
		return nil, err
	}
	aptManager := command.NewAptManager(ctx, runner)
	dockerManager := command.NewDockerManager(ctx, runner, aptManager)

	e.Ctx.Export("instance-ip", instance.PrivateIp)
	e.Ctx.Export("connection", conn)

	return &VM{Context: ctx, Runner: runner, Environment: e.CommonEnvironment, DockerManager: dockerManager}, nil
}
