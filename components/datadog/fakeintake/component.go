package fakeintake

import (
	"github.com/DataDog/test-infra-definitions/components"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type FakeintakeOutput struct {
	components.JSONImporter

	URL string `json:"url"`
}

type Fakeintake struct {
	pulumi.ResourceState
	components.Component

	// It's cleaner to export the full URL, but the Agent requires only host in some cases.
	// Keeping those internal to Pulumi program.
	Address pulumi.StringOutput
	Scheme  pulumi.StringOutput

	URL pulumi.StringOutput `pulumi:"url"`
}

func (fi *Fakeintake) Export(ctx *pulumi.Context, out *FakeintakeOutput) error {
	return components.Export(ctx, fi, out)
}
