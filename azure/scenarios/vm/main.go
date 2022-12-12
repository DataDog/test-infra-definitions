package main

import (
	"github.com/DataDog/test-infra-definitions/azure/scenarios/vm/vm"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(vm.Run)
}
