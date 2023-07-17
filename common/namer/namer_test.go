package namer

import (
	"fmt"
	"hash/fnv"
	"io"
	"math"
	"strconv"
	"strings"
	"testing"

	fuzz "github.com/google/gofuzz"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJoinWithMaxLength(t *testing.T) {
	for i, tt := range []struct {
		maxLength int
		tokens    []string
		noHash    bool
		expected  string
	}{
		// No truncation needed
		{
			maxLength: math.MaxInt,
			tokens:    []string{"foo", "bar", "baz"},
			expected:  "foo-bar-baz",
		},
		// Transition from full format to truncated format
		{
			maxLength: 20,
			tokens:    []string{"foo", "bar", "baz", "qux", "quux"},
			expected:  "foo-bar-baz-qux-quux",
		},
		{
			maxLength: 19,
			tokens:    []string{"foo", "bar", "baz", "qux", "quux"},
			expected:  "fo-bar-ba-qux-quu-1",
		},
		// Transition from truncated format to hash only
		{
			maxLength: 11,
			tokens:    []string{"foo", "bar", "baz", "qux", "quux"},
			expected:  "f-b-b-q-q-1",
		},
		{
			maxLength: 10,
			tokens:    []string{"foo", "bar", "baz", "qux", "quux"},
			expected:  "10087cd446",
		},
		// Max length too small
		// Defaults to hash only
		{
			maxLength: 4,
			tokens:    []string{"foo", "bar", "baz", "qux", "quux"},
			expected:  "1008",
		},
		// Truncation is applied to tokens proportionally to their size
		{
			maxLength: 23,
			tokens:    []string{"FfffOoooOooo", "BbbAaaRrr", "BbAaZz", "QUX"},
			noHash:    true,
			expected:  "FfffOooo-BbbAaa-BbAa-QU",
		},
		{
			maxLength: 13,
			tokens:    []string{"FfffOoooOooo", "BbbAaaRrr", "BbAaZz", "QUX"},
			noHash:    true,
			expected:  "Ffff-Bbb-Bb-Q",
		},
		// Truncation are spread best effort evenly on all tokens
		{
			maxLength: 18,
			tokens:    []string{"foo", "bar", "baz", "qux", "qux"},
			noHash:    true,
			expected:  "foo-bar-ba-qux-qux",
		},
		{
			maxLength: 17,
			tokens:    []string{"foo", "bar", "baz", "qux", "qux"},
			noHash:    true,
			expected:  "foo-ba-baz-qu-qux",
		},
		{
			maxLength: 16,
			tokens:    []string{"foo", "bar", "baz", "qux", "qux"},
			noHash:    true,
			expected:  "fo-bar-ba-qux-qu",
		},
		{
			maxLength: 15,
			tokens:    []string{"foo", "bar", "baz", "qux", "qux"},
			noHash:    true,
			expected:  "fo-ba-baz-qu-qu",
		},
		{
			maxLength: 14,
			tokens:    []string{"foo", "bar", "baz", "qux", "qux"},
			noHash:    true,
			expected:  "fo-ba-ba-qu-qu",
		},
		{
			maxLength: 13,
			tokens:    []string{"foo", "bar", "baz", "qux", "qux"},
			noHash:    true,
			expected:  "fo-ba-b-qu-qu",
		},
		{
			maxLength: 12,
			tokens:    []string{"foo", "bar", "baz", "qux", "qux"},
			noHash:    true,
			expected:  "fo-b-ba-q-qu",
		},
		{
			maxLength: 11,
			tokens:    []string{"foo", "bar", "baz", "qux", "qux"},
			noHash:    true,
			expected:  "f-ba-b-qu-q",
		},
		{
			maxLength: 10,
			tokens:    []string{"foo", "bar", "baz", "qux", "qux"},
			noHash:    true,
			expected:  "f-b-ba-q-q",
		},
		{
			maxLength: 9,
			tokens:    []string{"foo", "bar", "baz", "qux", "qux"},
			noHash:    true,
			expected:  "f-b-b-q-q",
		},
		// Some real cases with a stack name from the CI
		// 37 is the maximum size of EKS node group names
		{
			maxLength: 37,
			tokens:    []string{"ci-17317712-4670-eks-cluster", "linux", "ng"},
			expected:  "ci-17317712-4670-eks-cluster-linux-ng", // No truncation needed
		},
		{
			maxLength: 37,
			tokens:    []string{"ci-17317712-4670-eks-cluster", "linux-arm", "ng"},
			expected:  "ci-17317712-4670-eks-c-linux-a-ng-458",
		},
		{
			maxLength: 37,
			tokens:    []string{"ci-17317712-4670-eks-cluster", "bottlerocket", "ng"},
			expected:  "ci-17317712-4670-eks--bottleroc-n-a2e",
		},
		// 32 is the maximum size of load-balancer names
		{
			maxLength: 32,
			tokens:    []string{"ci-17317712-4670-eks-cluster", "fakeintake"},
			expected:  "ci-17317712-4670-eks-fakeint-5f1",
		},
		{
			maxLength: 32,
			tokens:    []string{"ci-17317712-4670-eks-cluster", "nginx"},
			expected:  "ci-17317712-4670-eks-cl-ngin-db3",
		},
		{
			maxLength: 32,
			tokens:    []string{"ci-17317712-4670-eks-cluster", "redis"},
			expected:  "ci-17317712-4670-eks-cl-redi-7de",
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			require.Conditionf(t, func() bool { return outputOK(tt.maxLength, tt.tokens, tt.expected) }, "Expected output string %q doesn’t match expected properties", tt.expected)
			noHash = tt.noHash
			output := joinWithMaxLength(tt.maxLength, tt.tokens)
			assert.Equal(t, tt.expected, output)
		})
	}
}

func outputOK(maxLength int, tokens []string, output string) bool {
	full := strings.Join(tokens, "-")
	if len(full) <= maxLength {
		return output == full
	}

	return len(output) == maxLength
}

func FuzzJoinWithMaxLength(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		f := fuzz.NewFromGoFuzz(data)

		var tt struct {
			maxLength int
			tokens    []string
		}
		f.Fuzz(&tt)

		output := joinWithMaxLength(tt.maxLength, tt.tokens)
		assert.Conditionf(t, func() bool { return outputOK(tt.maxLength, tt.tokens, output) }, "joinWithMaxLength(%d, %v) => %q", tt.maxLength, tt.tokens, output)
		assert.Equal(t, joinWithMaxLengthPrevImplem(tt.maxLength, tt.tokens), output)
	})
}

func BenchmarkJoinWithMaxLength(b *testing.B) {
	for _, tt := range []struct {
		name string
		f    func(int, []string) string
	}{
		{
			name: "New implementation",
			f:    joinWithMaxLength,
		},
		{
			name: "Old implementation",
			f:    joinWithMaxLengthPrevImplem,
		},
	} {
		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				joinWithMaxLength(37, []string{"ci-17317712-4670-eks-cluster", "linux-arm", "ng"})
			}
		})
	}
}

func joinWithMaxLengthPrevImplem(maxLength int, tokens []string) string {
	totalInputSize := lo.Sum(lo.Map(tokens, func(s string, _ int) int { return len(s) }))

	// Check if non-truncated concatenation fits inside maximum length
	if totalInputSize+(len(tokens)-1)*len(nameSep) <= maxLength {
		return strings.Join(tokens, nameSep)
	}

	// If a truncation is needed, a hash will be needed
	var fullhash string
	if !noHash {
		hasher := fnv.New64a()
		for _, tok := range tokens {
			_, _ = io.WriteString(hasher, tok)
		}
		fullhash = fmt.Sprintf("%016x", hasher.Sum64())
	}

	// Compute the size of the hash suffix that will be appended to the output
	hash := fullhash
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
		hashSize = -len(nameSep)
	}

	// If there’s so many tokens that truncating all of them to a single character string and keeping only the dash separators
	// would exceed the maximum length, we cannot do anything better than returning only the hash.
	if len(tokens)+(len(tokens))*len(nameSep)+hashSize > maxLength {
		if len(fullhash) > maxLength {
			return fullhash[:maxLength]
		}
		return fullhash
	}

	var sb strings.Builder

	// Truncate all tokens in the same relative proportion
	totalOutputSize := maxLength - len(tokens)*len(nameSep) - hashSize
	prevY := 0
	X := 0
	for _, tok := range tokens {
		X += len(tok)
		nextY := (2*X*totalOutputSize + totalInputSize) / (2 * totalInputSize)
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
