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
	IPAddress pulumi.StringInput
	stackKey  string
}

// ClientData client side data
type ClientData struct {
	IPAddress string
}

func (exporter *ConnectionExporter) Deserialize(result auto.UpResult) (*ClientData, error) {
	outputs, found := result.Outputs[exporter.stackKey]
	if !found {
		return nil, fmt.Errorf("cannot find %v in the stack result", exporter.stackKey)
	}
	ipAddress, ok := outputs.Value.(string)
	if !ok {
		return nil, fmt.Errorf("the type %v is not valid for the key %v", reflect.TypeOf(outputs.Value), exporter.stackKey)
	}
	return &ClientData{IPAddress: ipAddress}, nil
}

// NewExporter registers a fakeintake url into a Pulumi context.
func NewExporter(ctx *pulumi.Context, ipAddress pulumi.StringInput) *ConnectionExporter {
	ctx.Export(stackKey, ipAddress)
	return &ConnectionExporter{
		stackKey:  stackKey,
		IPAddress: ipAddress,
	}
}
