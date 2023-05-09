package utils

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// RemoteServiceConnector is a convenient struct to help implementing GetClientDataDeserializer().
// Here is an example of usage when you want to serialize a Connection and a string.
//
//	 type ClientData struct {
//	     MyString   string
//	     Connection utils.Connection
//	 }
//
//	 type MyRemoteService {
//	     // This implement GetClientDataDeserializer() func(auto.UpResult) (*ClientData, error)
//	     *RemoteServiceConnector[ClientData]
//	 }
//
//		remoteServiceConnector := utils.NewRemoteServiceConnector(ctx, ClientData{})
//		remoteServiceConnector.Register(stackKey, "Connection", connection)
//		remoteServiceConnector.Register("test", "MyString", pulumi.String("Hello"))
//
//		return &MyRemoteService{
//		    MyRemoteService: remoteServiceConnector
//		}
type RemoteServiceConnector[T any] struct {
	ctx                  *pulumi.Context
	stackKeyFieldNameMap map[string]string
	value                T
}

// NewRemoteServiceConnector creates a new instance of RemoteServiceConnector[T].
// initialValue is the initial value of the value returned by GetClientDataDeserializer.
// initialValue can be used to return non pulumi.Input type.
// T must be a struct.
func NewRemoteServiceConnector[T any](ctx *pulumi.Context, initialValue T) *RemoteServiceConnector[T] {
	return &RemoteServiceConnector[T]{
		ctx:                  ctx,
		value:                initialValue,
		stackKeyFieldNameMap: make(map[string]string)}
}

// Register registers a field.
// stackKeyName is the key in the stack name and should be unique
// outputFieldName is the name of the field in the struct
// value is the object to be saved in the stack
func (c *RemoteServiceConnector[T]) Register(stackKeyName string, outputFieldName string, value pulumi.Input) {
	c.ctx.Export(stackKeyName, value)
	c.stackKeyFieldNameMap[stackKeyName] = outputFieldName
}

func (c *RemoteServiceConnector[T]) Deserialize(upResult auto.UpResult) (*T, error) {
	value := reflect.ValueOf(&c.value).Elem()
	if value.Kind() != reflect.Struct {
		return nil, fmt.Errorf("the generic type T must be 'struct' whereas it is '%v'", value.Kind())
	}
	for stackKeyName, outputFieldName := range c.stackKeyFieldNameMap {
		field := value.FieldByName(outputFieldName)
		if !field.IsValid() {
			return nil, fmt.Errorf("the field '%v' of the struct '%v' doesn't exist", outputFieldName, value.Type())
		}

		if !field.CanSet() {
			return nil, fmt.Errorf("the field '%v' of the struct '%v' cannot be set", outputFieldName, value.Type())
		}

		outputs, found := upResult.Outputs[stackKeyName]
		if !found {
			return nil, fmt.Errorf("cannot find '%v' in the stack result", stackKeyName)
		}

		if field.Kind() == reflect.Struct {
			if err := deserializeStruct(field, outputs); err != nil {
				return nil, err
			}
		} else {
			field.Set(reflect.ValueOf(outputs.Value))
		}
	}
	return &c.value, nil
}

func deserializeStruct(field reflect.Value, output auto.OutputValue) error {
	oututFields := output.Value.(map[string]interface{})
	outputLowerFields := make(map[string]interface{})
	for k, v := range oututFields {
		outputLowerFields[strings.ToLower(k)] = v
	}
	fieldType := field.Type()
	for i := 0; i < field.NumField(); i++ {
		outputField := field.Field(i)
		outputFieldName := fieldType.Field(i).Name
		field, ok := outputLowerFields[strings.ToLower(outputFieldName)]
		if !ok {
			return fmt.Errorf("the field '%v' cannot be found in '%v'", outputFieldName, outputField.Type())
		}

		outputField.Set(reflect.ValueOf(field))
	}
	return nil
}
