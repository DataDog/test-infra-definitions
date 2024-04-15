package installer

import (
	"github.com/DataDog/test-infra-definitions/components/datadog/updater"
	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type installerLabVMArgs struct {
	name         string
	descriptor   os.Descriptor
	instanceType string
	packageNames []string
}

var installerLabVMs = []installerLabVMArgs{
	{
		name:         "ubuntu-22",
		descriptor:   os.NewDescriptorWithArch(os.Ubuntu, "22.04", os.ARM64Arch),
		instanceType: "t4g.medium",
		packageNames: []string{
			"datadog-agent",
		},
	},
	{
		name:         "debian-12",
		descriptor:   os.NewDescriptorWithArch(os.Debian, "12", os.ARM64Arch),
		instanceType: "t4g.medium",
		packageNames: []string{
			"datadog-agent",
		},
	},
	{
		name:         "amazon-linux-2023",
		descriptor:   os.NewDescriptorWithArch(os.AmazonLinux, "2023", os.ARM64Arch),
		instanceType: "t4g.medium",
		packageNames: []string{
			"datadog-agent",
		},
	},
}

func Run(ctx *pulumi.Context) error {
	env, err := aws.NewEnvironment(ctx)
	if err != nil {
		return err
	}

	for _, vmArgs := range installerLabVMs {
		vm, err := ec2.NewVM(
			env,
			vmArgs.name,
			ec2.WithInstanceType(vmArgs.instanceType),
			ec2.WithOSArch(vmArgs.descriptor, vmArgs.descriptor.Architecture),
		)
		if err != nil {
			return err
		}
		if err := vm.Export(ctx, nil); err != nil {
			return err
		}

		// Install the installer
		_, err = updater.NewHostUpdaterWithPackages(
			env.GetCommonEnvironment(), vm, vmArgs.packageNames,
		)
		if err != nil {
			return err
		}
	}

	return nil
}
