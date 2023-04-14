package fakeintake

import (
	"fmt"
	"reflect"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	stackKey = "fakeintake-url"
)

// PulumiData pulumi side data
type PulumiData struct {
	URL pulumi.StringInput
}

type exporter struct {
	stackKey string
}

// ClientData client side data
type ClientData struct {
	URL string
}

// GetClientDataDeserializer
func (exporter *exporter) GetClientDataDeserializer() func(auto.UpResult) (*ClientData, error) {
	return func(result auto.UpResult) (*ClientData, error) {
		outputs, found := result.Outputs[exporter.stackKey]
		if !found {
			return nil, fmt.Errorf("cannot find %v in the stack result", exporter.stackKey)
		}
		url, ok := outputs.Value.(string)
		if !ok {
			return nil, fmt.Errorf("the type %v is not valid for the key %v", reflect.TypeOf(outputs.Value), exporter.stackKey)
		}
		return &ClientData{URL: url}, nil
	}
}

type PulumiExporter struct {
	*exporter
	PulumiData
}

// NewExporter registers a fakeintake url into a Pulumi context.
func NewExporter(ctx *pulumi.Context, data PulumiData) *PulumiExporter {
	ctx.Export(stackKey, data.URL)
	return &PulumiExporter{
		exporter: &exporter{
			stackKey: stackKey,
		},
		PulumiData: data,
	}
}
