package common

import (
	"strings"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const nameSep = "-"

type Namer struct {
	ctx    *pulumi.Context
	prefix string
}

func NewNamer(ctx *pulumi.Context, prefix string) Namer {
	return Namer{
		ctx:    ctx,
		prefix: prefix,
	}
}

func (n Namer) WithPrefix(prefix string) Namer {
	childNamer := Namer{}
	if n.prefix != "" {
		childNamer.prefix = n.prefix + nameSep
	}
	childNamer.prefix += prefix
	return childNamer
}

func (n Namer) ResourceName(parts ...string) string {
	if len(parts) == 0 {
		panic("Resource name requires at least one part to generate name")
	}

	var resourceName string
	if n.prefix != "" {
		resourceName += n.prefix + nameSep
	}

	return resourceName + strings.Join(parts, nameSep)
}

func (n Namer) DisplayName(parts ...pulumi.StringInput) pulumi.StringInput {
	var convertedParts []interface{}
	for _, part := range parts {
		convertedParts = append(convertedParts, part)
	}
	return pulumi.All(convertedParts...).ApplyT(func(args []interface{}) string {
		strArgs := []string{n.ctx.Stack()}
		for _, arg := range args {
			strArgs = append(strArgs, arg.(string))
		}
		return n.ResourceName(strArgs...)
	}).(pulumi.StringOutput)
}
