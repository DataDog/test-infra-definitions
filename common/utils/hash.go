package utils

import (
	"fmt"
	"hash/fnv"
	"io"
	"os"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
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

func PulumiStrHash(all ...pulumi.StringInput) pulumi.StringInput {
	return pulumi.All(all).ApplyT(func(args []interface{}) string {
		allStr := make([]string, len(args))
		for i, arg := range args {
			allStr[i] = fmt.Sprintf("%s", arg)
		}
		return StrHash(allStr...)
	}).(pulumi.StringOutput)
}
