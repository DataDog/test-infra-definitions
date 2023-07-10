package namer

import (
	"math"
	"strconv"
	"strings"
	"testing"

	fuzz "github.com/google/gofuzz"
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
		// Truncation are spread evenly on all tokens
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
		assert.Condition(t, func() bool { return outputOK(tt.maxLength, tt.tokens, output) }, "joinWithMaxLength(%d, %v) => %q", tt.maxLength, tt.tokens, output)
	})
}
