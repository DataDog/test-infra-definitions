package vm

import (
	"fmt"

	ec2vm "github.com/DataDog/test-infra-definitions/aws/scenarios/vm/ec2VM"
	ec2os "github.com/DataDog/test-infra-definitions/aws/scenarios/vm/os"
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/datadog/agent"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Run(ctx *pulumi.Context) error {
	env := config.NewCommonEnvironment(ctx)
	var osType ec2os.Type

	osTypeStr := env.InfraOSType()
	switch osTypeStr {
	case "Windows":
		osType = ec2os.WindowsOS
	case "Ubuntu":
		osType = ec2os.UbuntuOS
	case "AmazonLinux":
		osType = ec2os.AmazonLinuxOS
	case "Debian":
		osType = ec2os.DebianOS
	case "RedHat":
		osType = ec2os.RedHatOS
	case "Suse":
		osType = ec2os.SuseOS
	case "Fedora":
		osType = ec2os.FedoraOS
	default:
		return fmt.Errorf("the os type '%v' is not valid", osTypeStr)
	}

	vm, err := ec2vm.NewEc2VM(ctx, ec2vm.WithOS(osType))
	if err != nil {
		return err
	}

	if vm.GetCommonEnvironment().AgentDeploy() {
		_, err = agent.NewInstaller(vm)
		return err
	}

	return nil
}
