package main

import (
	"github.com/DataDog/test-infra-definitions/aws/ecs/ecs"
	"github.com/DataDog/test-infra-definitions/common"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		return common.Run(ctx, ecs.Run)
	})
}
