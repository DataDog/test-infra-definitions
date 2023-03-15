package utils

import (
	"fmt"
	"reflect"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
)

type Connection struct {
	Host string
	User string
}

func NewConnection(result auto.UpResult, key string) (Connection, error) {
	outputs, found := result.Outputs[key]
	if !found {
		return Connection{}, fmt.Errorf("cannot find %v in the stack result", key)
	}

	values, ok := outputs.Value.(map[string]interface{})
	if !ok {
		return Connection{}, fmt.Errorf("the type %v is not valid for the key %v", reflect.TypeOf(outputs.Value), key)
	}

	host, err := getMapStringValue(values, "host")
	if err != nil {
		return Connection{}, err
	}

	user, err := getMapStringValue(values, "user")
	if err != nil {
		return Connection{}, err
	}

	return Connection{
		Host: host,
		User: user,
	}, nil

}

func getMapStringValue(values map[string]interface{}, key string) (string, error) {
	value, found := values[key]
	if !found {
		return "", fmt.Errorf("key %v not found", key)
	}
	result, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("invalid type for %v: %v. It must be `string`", key, reflect.TypeOf(key))
	}
	return result, nil
}
