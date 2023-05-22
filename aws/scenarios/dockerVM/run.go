package dockervm

import (
	ec2vm "github.com/DataDog/test-infra-definitions/aws/scenarios/vm/ec2VM"
	"github.com/DataDog/test-infra-definitions/datadog/agent/docker"

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
