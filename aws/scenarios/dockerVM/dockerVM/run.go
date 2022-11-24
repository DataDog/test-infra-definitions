package dockerVM

import (
	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/aws/ec2/ec2"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Run(ctx *pulumi.Context) error {
	e, err := aws.AWSEnvironment(ctx)
	if err != nil {
		return err
	}

	if e.AgentDeploy() {
		return DeployWithAgent(ctx, "")
	}

	instance, conn, err := ec2.NewDefaultEC2Instance(e, ctx.Stack(), e.DefaultInstanceType())
	if err != nil {
		return err
	}

	e.Ctx.Export("instance-ip", instance.PrivateIp)
	e.Ctx.Export("connection", conn)

	return nil
}
