package main

import (
	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/aws/ec2/ec2"
	"github.com/DataDog/test-infra-definitions/datadog/agent"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		e, err := aws.AWSEnvironment(ctx)
		if err != nil {
			return err
		}

		instance, conn, err := ec2.NewDefaultEC2Instance(e, ctx.Stack(), e.DefaultInstanceType())
		if err != nil {
			return err
		}

		if e.AgentDeploy() {
			_, err := agent.NewHostInstallation(*e.CommonEnvironment, ctx.Stack(), conn)
			if err != nil {
				return err
			}
		}

		e.Ctx.Export("instance-ip", instance.PrivateIp)
		e.Ctx.Export("connection", conn)

		return nil
	})
}
