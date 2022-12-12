package vm

import (
	"github.com/DataDog/test-infra-definitions/azure"
	"github.com/DataDog/test-infra-definitions/azure/compute"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Run(ctx *pulumi.Context) error {
	env, err := azure.AzureEnvironment(ctx)
	if err != nil {
		return err
	}

	_, publicIP, nw, adminPassword, err := compute.NewWindowsInstance(env, "vm", compute.WindowsLatestURN(), env.DefaultInstanceType(), nil, nil)
	if err != nil {
		return err
	}

	ctx.Export("private-instance-ip", nw.IpConfigurations.Index(pulumi.Int(0)).PrivateIPAddress())
	ctx.Export("public-instance-ip", publicIP.IpAddress)
	ctx.Export("admin-password", adminPassword)
	return nil
}
