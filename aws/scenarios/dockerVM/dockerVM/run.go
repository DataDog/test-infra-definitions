package dockerVM

import (
	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/aws/ec2/ec2"
	"github.com/DataDog/test-infra-definitions/command"
	"github.com/DataDog/test-infra-definitions/datadog/agent"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Run(ctx *pulumi.Context) error {
	e, err := aws.AWSEnvironment(ctx)
	if err != nil {
		return err
	}

	instance, conn, err := ec2.NewDefaultEC2Instance(e, ctx.Stack(), e.DefaultInstanceType())
	if err != nil {
		return err
	}

	if e.AgentDeploy() {
		runner, err := command.NewRunner(ctx.Stack()+"-conn", conn, func(r *command.Runner) (*remote.Command, error) {
			return command.WaitForCloudInit(ctx, r)
		})
		if err != nil {
			return err
		}

		aptManager := command.NewAptManager(e.Ctx, runner)
		dockerManager := command.NewDockerManager(e.Ctx, runner, aptManager)
		_, err = agent.NewDockerInstallation(*e.CommonEnvironment, dockerManager, nil)
		if err != nil {
			return err
		}
	}

	e.Ctx.Export("instance-ip", instance.PrivateIp)
	e.Ctx.Export("connection", conn)

	return nil
}
