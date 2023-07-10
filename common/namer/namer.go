package namer

import (
	"fmt"
	"hash/fnv"
	"io"
	"math"
	"strings"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/samber/lo"
)

const nameSep = "-"

// In case of truncation, size of the hash suffix in percentage of the full size
const hashSizePercent = 10

// Used only for tests to skip hash suffix to ease test results validation
var noHash = false

type Namer struct {
	ctx      *pulumi.Context
	prefixes []string
}

func NewNamer(ctx *pulumi.Context, prefix string) Namer {
	if prefix == "" {
		return Namer{
			ctx:      ctx,
			prefixes: []string{},
		}
	}
	return Namer{
		ctx:      ctx,
		prefixes: []string{prefix},
	}
}

func (n Namer) WithPrefix(prefix string) Namer {
	return Namer{
		ctx:      n.ctx,
		prefixes: append(n.prefixes, prefix),
	}
}

func (n Namer) ResourceName(parts ...string) string {
	if len(parts) == 0 {
		panic("Resource name requires at least one part to generate name")
	}

	return joinWithMaxLength(math.MaxInt, append(n.prefixes, parts...))
}

func (n Namer) DisplayName(maxLen int, parts ...pulumi.StringInput) pulumi.StringInput {
	var convertedParts []interface{}
	for _, part := range parts {
		convertedParts = append(convertedParts, part)
	}
	return pulumi.All(convertedParts...).ApplyT(func(args []interface{}) string {
		strArgs := make([]string, 1, 1+len(n.prefixes)+len(args))
		strArgs[0] = n.ctx.Stack()
		strArgs = append(strArgs, n.prefixes...)
		for _, arg := range args {
			strArgs = append(strArgs, arg.(string))
		}
		return joinWithMaxLength(maxLen, strArgs)
	}).(pulumi.StringOutput)
}

func joinWithMaxLength(maxLength int, tokens []string) string {
	// Check if non-truncated concatenation fits inside maximum length
	if lo.Sum(lo.Map(tokens, func(s string, _ int) int { return len(s) }))+(len(tokens)-1)*len(nameSep) <= maxLength {
		return strings.Join(tokens, nameSep)
	}

	// If a truncation is needed, a hash will be needed
	hasher := fnv.New64a()
	for _, tok := range tokens {
		_, _ = io.WriteString(hasher, tok)
	}
	hash := fmt.Sprintf("%x", hasher.Sum64())

	// If there’s so many tokens that truncating all of them to empty string and keeping only the dash separators
	// would exceed the maximum length, we cannot do anything better than returning only the hash.
	if (len(tokens)-1)*len(nameSep) >= maxLength {
		if len(hash) > maxLength {
			return hash[:maxLength]
		}
		return hash
	}

	// Compute the size of the hash suffix that will be appended to the output
	hashSize := maxLength * hashSizePercent / 100
	if len(hash) > hashSize {
		hash = hash[:hashSize]
	} else {
		hashSize = len(hash)
	}

	// For test purpose, we have an option to completely strip the output of the hash suffix
	// At this point, `hashSize` is the size of the hash suffix without the dash separator
	// -1 means that we want to also strip the dash
	if noHash {
		hashSize = -1
	}

	var sb strings.Builder

	// Truncate all tokens in the same relative proportion
	totalOutputSize := maxLength - len(tokens)*len(nameSep) - hashSize
	totalInputSize := lo.Sum(lo.Map(tokens, func(s string, _ int) int { return len(s) }))
	prevY := 0
	X := 0
	for _, tok := range tokens {
		X += len(tok)
		nextY := (X*totalOutputSize + totalInputSize/2) / totalInputSize
		tokSize := nextY - prevY
		prevY = nextY
		sb.WriteString(tok[:tokSize])
		sb.WriteString(nameSep)
	}

	if noHash {
		str := sb.String()
		return str[:len(str)-1] // Strip the trailing dash
	}

	sb.WriteString(hash)
	return sb.String()
}
