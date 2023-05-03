package vm

import (
	"fmt"
	"strings"

	ec2vm "github.com/DataDog/test-infra-definitions/aws/scenarios/vm/ec2VM"
	ec2os "github.com/DataDog/test-infra-definitions/aws/scenarios/vm/os"
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/datadog/agent"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Run(ctx *pulumi.Context) error {
	env := config.NewCommonEnvironment(ctx)
	var osType ec2os.Type

	osTypeStr := strings.ToLower(env.InfraOSFamily())
	switch osTypeStr {
	case "windows":
		osType = ec2os.WindowsOS
	case "ubuntu":
		osType = ec2os.UbuntuOS
	case "amazonlinux":
		osType = ec2os.AmazonLinuxOS
	case "debian":
		osType = ec2os.DebianOS
	case "redhat":
		osType = ec2os.RedHatOS
	case "suse":
		osType = ec2os.SuseOS
	case "fedora":
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
