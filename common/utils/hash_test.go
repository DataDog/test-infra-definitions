package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_StrUniqueWithMaxLen(t *testing.T) {
	t.Run("should return the original string when it is not longer than max len", func(t *testing.T) {
		s := "totoro"
		maxLen := 6
		bestEffortHash := StrUniqueWithMaxLen(s, maxLen)
		assert.Equal(t, "totoro", bestEffortHash)
		assert.Equal(t, maxLen, len(bestEffortHash))
	})

	t.Run("should return first chars of the original string plus 3 chars from the hash when it is not longer than max len", func(t *testing.T) {
		s := "totoro"
		maxLen := 5
		bestEffortHash := StrUniqueWithMaxLen(s, maxLen)
		assert.Equal(t, maxLen, len(bestEffortHash))
		assert.Equal(t, "t-8bd", bestEffortHash)
	})

	t.Run("should return a readable name with an hash when passing a long string", func(t *testing.T) {
		s := "user-longlonglonglonglongname-aws-vm-test-with-totoro"
		maxLen := 32
		bestEffortHash := StrUniqueWithMaxLen(s, maxLen)
		assert.Equal(t, maxLen, len(bestEffortHash))
		assert.Equal(t, "user-longlonglonglonglongnam-47d", bestEffortHash)
	})
}
