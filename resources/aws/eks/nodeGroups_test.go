package eks

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNodeGroupNamePrefix(t *testing.T) {
	for i, tt := range []struct {
		stack    string
		name     string
		expected string
	}{
		{
			stack:    "short",
			name:     "short",
			expected: "short-short-ng",
		},
		{
			stack:    "ci-17317712-4670-eks-cluster",
			name:     "linux",
			expected: "ci-17317712-4670-eks-cluster-linux-ng",
		},
		{
			stack:    "ci-17317712-4670-eks-cluster",
			name:     "linux-arm",
			expected: "ci-17317712-4670-eks-clus-linux-ar-ng",
		},
		{
			stack:    "ci-17317712-4670-eks-cluster",
			name:     "bottlerocket",
			expected: "ci-17317712-4670-eks-clu-bottleroc-ng",
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			prefix := nodeGroupNamePrefix(tt.stack, tt.name)
			assert.Equal(t, tt.expected, prefix)
			assert.LessOrEqual(t, len(prefix), 37)
		})
	}
}

func FuzzNodeGroupNamePrefix(f *testing.F) {
	f.Add("short", "short")
	f.Add("ci-17317712-4670-eks-cluster", "linux")
	f.Add("ci-17317712-4670-eks-cluster", "linux-arm")
	f.Add("ci-17317712-4670-eks-cluster", "bottlerocket")
	f.Add("crazy-long-stack-name-crazy-long-stack-name", "x")
	f.Add("x", "crazy-long-node-group-name-crazy-long-node-group-name")

	f.Fuzz(func(t *testing.T, stack, name string) {
		untruncatedPrefix := stack + "-" + name + "-ng"
		prefix := nodeGroupNamePrefix(stack, name)
		assert.Conditionf(t, func() bool {
			if len(untruncatedPrefix) <= 37 {
				return prefix == untruncatedPrefix
			}
			return len(prefix) == 37
		}, "nodeGroupNamePrefix(%q, %q) => %q (len=%d)", stack, name, prefix, len(prefix))
	})
}
