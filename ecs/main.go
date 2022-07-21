package main

import (
	"github.com/vboulineau/pulumi-definitions/ecs/ecs"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(ecs.Run)
}
