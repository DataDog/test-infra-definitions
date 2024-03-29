package fakeintake

import (
	"github.com/DataDog/test-infra-definitions/components"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type FakeintakeOutput struct { // nolint:revive, We want to keep the name as <Component>Output
	components.JSONImporter

	URL string `json:"url"`
}

type Fakeintake struct {
	pulumi.ResourceState
	components.Component

	// It's cleaner to export the full URL, but the Agent requires only host in some cases.
	// Keeping those internal to Pulumi program.
	Host   pulumi.StringOutput
	Scheme string // Scheme is a string as it's known in code and is useful to check HTTP/HTTPS
	Port   uint32 // Same for Port

	URL pulumi.StringOutput `pulumi:"url"`
}

func (fi *Fakeintake) Export(ctx *pulumi.Context, out *FakeintakeOutput) error {
	return components.Export(ctx, fi, out)
}
