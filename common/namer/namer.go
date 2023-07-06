package namer

import (
	"crypto/md5"
	"encoding/hex"
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

// HashedName return the md5 32-chars hex pulumi.StringInput of `parts` prefixed
// with the namer prefix
func (n Namer) HashedName(parts ...pulumi.StringInput) pulumi.StringInput {
	return n.DisplayName(parts...).ToStringOutput().ApplyT(func(name string) string {
		hmd5 := md5.Sum([]byte(name))
		return hex.EncodeToString(hmd5[:])
	}).(pulumi.StringOutput)
}

// DisplayHashedName return the first 15 chars of the display name plus the first 16 chars
// of the md5 32-chars hex pulumi.StringInput of display name
func (n Namer) DisplayHashedName(parts ...pulumi.StringInput) pulumi.StringInput {
	return pulumi.All(n.DisplayName(parts...), n.HashedName(parts...)).ApplyT(func(args []interface{}) string {
		displayName := args[0].(string)
		hashedName := args[1].(string)
		return strings.Join([]string{displayName[:15], hashedName[:16]}, "-")
	}).(pulumi.StringOutput)
}
