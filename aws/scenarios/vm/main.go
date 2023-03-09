package main

import (
	ec2vm "github.com/DataDog/test-infra-definitions/aws/scenarios/vm/ec2VM"
	"github.com/DataDog/test-infra-definitions/aws/scenarios/vm/os"
	"github.com/DataDog/test-infra-definitions/common/config"
	commonos "github.com/DataDog/test-infra-definitions/common/os"
	"github.com/DataDog/test-infra-definitions/datadog/agent"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		vm, err := ec2vm.NewUnixEc2VM(ctx, ec2vm.WithOS(os.AmazonLinuxOS, commonos.AMD64Arch))

		commonEnv := config.NewCommonEnvironment(ctx)
		if commonEnv.AgentConfig.GetBool("deploy") {
			_, err = agent.NewInstaller(vm)
		}
		return err
	})
}
