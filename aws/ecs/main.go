package main

import (
	"github.com/vboulineau/pulumi-definitions/aws/ecs/ecs"
	"github.com/vboulineau/pulumi-definitions/common"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		return common.Run(ctx, ecs.Run)
	})
}
