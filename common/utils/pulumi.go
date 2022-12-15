package utils

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func PulumiDependsOn(resources ...pulumi.Resource) pulumi.ResourceOption {
	return pulumi.DependsOn(resources)
}
