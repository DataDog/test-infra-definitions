package fakeintake

import (
	"fmt"
	"reflect"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	stackKey = "fakeintake-ip-address"
)

// ConnectionExporter contains pulumi side data and the export key
type ConnectionExporter struct {
	Host     pulumi.StringInput
	stackKey string
}

// ClientData client side data
type ClientData struct {
	Host string
}

func (exporter *ConnectionExporter) Deserialize(result auto.UpResult) (*ClientData, error) {
	outputs, found := result.Outputs[exporter.stackKey]
	if !found {
		return nil, fmt.Errorf("cannot find %v in the stack result", exporter.stackKey)
	}
	host, ok := outputs.Value.(string)
	if !ok {
		return nil, fmt.Errorf("the type %v is not valid for the key %v", reflect.TypeOf(outputs.Value), exporter.stackKey)
	}
	return &ClientData{Host: host}, nil
}

// NewExporter registers a fakeintake url into a Pulumi context.
func NewExporter(ctx *pulumi.Context, host pulumi.StringInput) *ConnectionExporter {
	ctx.Export(stackKey, host)
	return &ConnectionExporter{
		stackKey: stackKey,
		Host:     host,
	}
}
