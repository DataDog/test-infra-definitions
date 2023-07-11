package utils

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func PulumiDependsOn(resources ...pulumi.Resource) pulumi.ResourceOption {
	return pulumi.DependsOn(resources)
}

func MergeOptions[T any](current []T, opts ...T) []T {
	if len(opts) == 0 {
		return current
	}

	addedOptions := make([]T, len(current)+len(opts))
	for _, array := range [][]T{current, opts} {
		for _, opt := range array {
			addedOptions = append(addedOptions, opt)
		}
	}

	return addedOptions
}
