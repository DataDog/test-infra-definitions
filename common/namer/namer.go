package namer

import (
	"strings"

	"github.com/DataDog/test-infra-definitions/common/utils"
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

// ResourceName return the concatenation of `parts` prefixed
// with namer prefix
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

// DisplayName return pulumi.StringInput the concatanation of pulumi.StringInput `parts` prefixed
// with the namer prefix
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

// DisplayNameWithMaxLen return pulumi.StringInput the concatanation of pulumi.StringInput `parts` prefixed
// with the namer prefix
func (n Namer) DisplayNameWithMaxLen(maxLen int, parts ...pulumi.StringInput) pulumi.StringInput {
	return n.DisplayName(parts...).ToStringOutput().ApplyT(func(s string) string {
		return utils.StrUniqueWithMaxLen(s, maxLen)
	}).(pulumi.StringOutput)
}
