package main

import (
	microVM "github.com/DataDog/test-infra-definitions/aws/scenarios/microVMs/microVMs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(microVM.Run)
}
