package main

import (
	"github.com/DataDog/test-infra-definitions/azure/scenarios/aks/aks"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(aks.Run)
}
