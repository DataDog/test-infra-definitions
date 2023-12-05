package fakeintake

import (
	"github.com/DataDog/test-infra-definitions/components"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type FakeintakeOutput struct {
	components.JSONImporter

	Address string `json:"address"`
}

type Fakeintake struct {
	pulumi.ResourceState
	components.Component

	Address pulumi.StringOutput `pulumi:"address"`
}

func (fi *Fakeintake) Export(ctx *pulumi.Context, out *FakeintakeOutput) error {
	return components.Export(ctx, fi, out)
}
