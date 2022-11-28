package main

import (
	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/aws/ec2/ec2"
	"github.com/DataDog/test-infra-definitions/command"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		e, err := aws.AWSEnvironment(ctx)
		if err != nil {
			return err
		}

		// boot aws metal instance
		instance, conn, err := ec2.NewDefaultEC2Instance(e, ctx.Stack(), e.DefaultInstanceType())
		if err != nil {
			return err
		}

		// install qemu-kvm and libvirt-daemon-system
		runner, err := command.NewRunner(ctx.Stack()+"-conn", conn, func(r *command.Runner) (*remote.Command, error) {
			return command.WaitForCloudInit(ctx, r)
		})
		if err != nil {
			return err
		}

		aptManager := command.NewAptManager(e.Ctx, runner)
		installQemu, err := aptManager.Ensure("qemu-kvm")
		if err != nil {
			return err
		}
		installLibVirt, err := aptManager.Ensure("libvirt-daemon-system", pulumi.DependsOn([]pulumi.Resource{installQemu}))
		if err != nil {
			return err
		}

		_, err = runner.Command(ctx, "libvirt-group", pulumi.String("sudo adduser $USER libvirt"), nil, nil, nil, true, pulumi.DependsOn([]pulumi.Resource{installLibVirt}))

		e.Ctx.Export("instance-ip", instance.PrivateIp)
		e.Ctx.Export("connection", conn)

		return nil
	})
}
