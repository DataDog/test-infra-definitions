package main

import (
	dockervm "github.com/DataDog/test-infra-definitions/aws/scenarios/dockerVM/dockerVM"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(dockervm.Run)
}
