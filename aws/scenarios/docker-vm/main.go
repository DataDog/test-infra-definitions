package main

import (
	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/aws/ec2/ec2"
	"github.com/DataDog/test-infra-definitions/command"
	"github.com/DataDog/test-infra-definitions/datadog/agent"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		e, err := aws.AWSEnvironment(ctx)
		if err != nil {
			return err
		}

		instance, conn, err := ec2.NewDefaultEC2Instance(e, "docker-vm", e.DefaultInstanceType())
		if err != nil {
			return err
		}

		if e.AgentDeploy() {
			runner, err := command.NewRunner(*e.CommonEnvironment, e.CommonNamer.ResourceName("docker-vm"), conn, func(r *command.Runner) (*remote.Command, error) {
				return command.WaitForCloudInit(ctx, r)
			})
			if err != nil {
				return err
			}

			aptManager := command.NewAptManager(runner)
			dockerManager := command.NewDockerManager(runner, aptManager)
			_, err = agent.NewDockerAgentInstallation(e.CommonEnvironment, dockerManager, "", "")
			if err != nil {
				return err
			}
		}

		e.Ctx.Export("instance-ip", instance.PrivateIp)
		e.Ctx.Export("connection", conn)

		return nil
	})
}
