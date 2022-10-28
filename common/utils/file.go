package utils

import (
	"crypto/sha256"
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

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return string(h.Sum(nil)), nil
}

func StrHash(s string) string {
	h := sha256.New()
	io.WriteString(h, s)

	return string(h.Sum(nil))
}

func ReadSecretFile(filePath string) (pulumi.StringPtrOutput, error) {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return pulumi.StringPtrOutput{}, err
	}

	s := pulumi.ToSecret(pulumi.StringPtr(string(b))).(pulumi.StringPtrOutput)

	return s, nil
}

func WriteStringCommand(content pulumi.StringInput, filePath string) pulumi.StringInput {
	return pulumi.Sprintf(`cat <<EOF > %s
%s
EOF`, filePath, content)
}
