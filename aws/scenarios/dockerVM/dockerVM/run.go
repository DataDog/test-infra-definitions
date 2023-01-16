package dockerVM

import (
	"fmt"

	ec2vm "github.com/DataDog/test-infra-definitions/aws/scenarios/vm/ec2VM"
	"github.com/DataDog/test-infra-definitions/common/docker"
	commonvm "github.com/DataDog/test-infra-definitions/common/vm"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Run(ctx *pulumi.Context) error {
	vm, err := ec2vm.NewEc2VM(ctx)
	if err != nil {
		return err
	}
	ubuntuVM, ok := vm.(*commonvm.UbuntuVM)
	if !ok {
		return fmt.Errorf("not an ubuntu VM")
	}
	_, err = docker.NewDockerOnVM(ctx, ubuntuVM, docker.WithAgent())

	return err
}
