package dockervm

import (
	"github.com/DataDog/test-infra-definitions/components/datadog/agent/docker"
	"github.com/DataDog/test-infra-definitions/components/datadog/agent/dockerparams"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/vm/ec2vm"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Run(ctx *pulumi.Context) error {
	vm, err := ec2vm.NewUnixEc2VM(ctx)
	if err != nil {
		return err
	}

	env := vm.GetCommonEnvironment()

	var options []dockerparams.Option
	if env.AgentDeploy() {
		options = append(options, dockerparams.WithAgent())
	}

	_, err = docker.NewAgentDockerInstaller(vm.UnixVM, options...)

	return err
}
