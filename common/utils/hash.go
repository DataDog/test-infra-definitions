package utils

import (
	"fmt"
	"hash/fnv"
	"io"
	"os"
)

func FileHash(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := fnv.New64a()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum64()), nil
}

func StrHash(all ...string) string {
	h := fnv.New64a()
	for _, s := range all {
		_, _ = io.WriteString(h, s)
	}

	return fmt.Sprintf("%x", h.Sum64())
}

// StrUniqueWithMaxLen returns a best effort readable and unique string given a max size.
// If the string is not longer than maxLen it returns the original string. If the string is
// longer it returns (maxLen - 4) chars of the original string plus 3 chars (1 byte and 1 word)
// of the hash of the string.
// Example: ("totoro", 5) => "t-8bd"
func StrUniqueWithMaxLen(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	hash := StrHash(s)
	if maxLen <= 4 {
		return hash[:maxLen]
	}
	prefix := s[:maxLen-4] + "-"
	return fmt.Sprintf("%s%s", prefix, hash[:3])
}
