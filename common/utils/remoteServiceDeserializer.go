package utils

import "github.com/pulumi/pulumi/sdk/v3/go/auto"

type RemoteServiceDeserializer[T any] interface {
	Deserialize(auto.UpResult) (*T, error)
}
