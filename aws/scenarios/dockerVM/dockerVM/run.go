package dockervm

import (
	ec2vm "github.com/DataDog/test-infra-definitions/aws/scenarios/vm/ec2VM"
	"github.com/DataDog/test-infra-definitions/datadog/agent/docker"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Run(ctx *pulumi.Context) error {
	vm, err := ec2vm.NewUnixLikeEc2VM(ctx)
	if err != nil {
		return err
	}

	_, err = docker.NewAgentDockerInstaller(vm, docker.WithAgent())

	return err
}
