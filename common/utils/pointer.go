package utils

import "github.com/pulumi/pulumi/sdk/v3/go/pulumi"

func Pointer[T any](value T) *T {
	return &value
}

func StringPtr(s string) pulumi.StringPtrInput {
	if len(s) > 0 {
		return pulumi.StringPtr(s)
	}

	return nil
}
