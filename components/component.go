package components

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Importable needs to be implemented by the fully resolved type used outside of Pulumi
type Importable interface {
	SetKey(string)
	Key() string
	Import(in []byte, obj any) error
}

var _ Importable = &JSONImporter{}

type JSONImporter struct {
	key string
}

func (imp *JSONImporter) SetKey(key string) {
	imp.key = key
}

func (imp *JSONImporter) Key() string {
	return imp.key
}

func (imp *JSONImporter) Import(in []byte, obj any) error {
	return json.Unmarshal(in, obj)
}

type component interface {
	pulumi.ComponentResource

	init(name string, exportName string)
	getOutputs() pulumi.Map
	getExportName() string
	registerOutputs(ctx *pulumi.Context, self pulumi.ComponentResource) error
}

type Component struct {
	name       string // Name is set to the name of Pulumi component, it allows to name dependencies easily.
	outputs    pulumi.Map
	exportName string
}

func (c *Component) init(name, exportName string) {
	c.name = name
	c.outputs = make(pulumi.Map)
	c.exportName = exportName
}

func (c *Component) getOutputs() pulumi.Map { //nolint:unused, used through the `component` interface
	return c.outputs
}

func (c *Component) getExportName() string { //nolint:unused, used through the `component` interface
	return c.exportName
}

func (c *Component) Name() string {
	return c.name
}

// RegisterOutputs exports values from a `pulumi.ComponentResource`. Use `pulumi` tag to export a field.
func (c *Component) registerOutputs(ctx *pulumi.Context, self pulumi.ComponentResource) error {
	fields := reflect.VisibleFields(reflect.TypeOf(self).Elem())
	compValue := reflect.ValueOf(self).Elem()
	for _, field := range fields {
		if exportFieldName := field.Tag.Get("pulumi"); exportFieldName != "" {
			if !field.IsExported() {
				continue
			}

			if !field.Type.Implements(reflect.TypeOf((*pulumi.Input)(nil)).Elem()) {
				return fmt.Errorf("trying to export a field that is not a pulumi.Output, field name: %s", field.Name)
			}

			if _, set := c.outputs[exportFieldName]; set {
				return fmt.Errorf("cannot export field: %s as key %s is already used", field.Name, exportFieldName)
			}

			if fieldValue := compValue.FieldByIndex(field.Index).Interface(); fieldValue != nil {
				c.outputs[exportFieldName] = fieldValue.(pulumi.Input)
			}
		}
	}

	return ctx.RegisterResourceOutputs(self, c.outputs)
}

// Export should not be used directly but only by an `Importable` type itself to add type safety.
func Export(ctx *pulumi.Context, c component, imp Importable) error {
	// To reproduce the current cross-program assignment in `datadog-agent`, not technically required.
	if imp != nil && !reflect.ValueOf(imp).IsNil() {
		imp.SetKey(c.getExportName())
	}

	ctx.Export(c.getExportName(), c.getOutputs().ToMapOutput())
	return nil
}

// Create any component type and register it as a Pulumi component
// Passing a nil `builder` is valid and will only produce an empty component.
func NewComponent[C component](e config.CommonEnvironment, name string, builder func(comp C) error, opts ...pulumi.ResourceOption) (C, error) {
	var comp C

	compType := reflect.TypeOf(comp)
	if compType.Kind() != reflect.Pointer {
		return comp, fmt.Errorf("component type: %T is not pointer, cannot allocate", comp)
	}

	compName := reflect.TypeOf(comp).Elem().Name()
	comp = reflect.New(compType.Elem()).Interface().(C)

	comp.init(name, e.CommonNamer.ResourceName("dd", compName, name))
	err := e.Ctx.RegisterComponentResource(fmt.Sprintf("dd:%s", compName), e.CommonNamer.ResourceName(name), comp, opts...)
	if err != nil {
		return comp, err
	}

	if builder != nil {
		err = builder(comp)
		if err != nil {
			return comp, err
		}
	}

	return comp, comp.registerOutputs(e.Ctx, comp)
}
