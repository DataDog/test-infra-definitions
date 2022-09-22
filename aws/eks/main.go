package main

import (
	"github.com/DataDog/test-infra-definitions/aws/eks/eks"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		return eks.Run(ctx)
	})
}
