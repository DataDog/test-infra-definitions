package dockervm

import (
	"github.com/DataDog/test-infra-definitions/components/datadog/agent/docker"
	ec2vm "github.com/DataDog/test-infra-definitions/scenarios/aws/vm/ec2VM"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Run(ctx *pulumi.Context) error {
	vm, err := ec2vm.NewUnixEc2VM(ctx)
	if err != nil {
		return err
	}

	env := vm.GetCommonEnvironment()

	var options []func(*docker.Params) error
	if env.AgentDeploy() {
		options = append(options, docker.WithAgent())
	}

	_, err = docker.NewAgentDockerInstaller(vm.UnixVM, options...)

	return err
}
