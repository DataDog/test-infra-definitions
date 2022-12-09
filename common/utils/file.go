package utils

import (
	"os"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func ReadSecretFile(filePath string) (pulumi.StringPtrOutput, error) {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return pulumi.StringPtrOutput{}, err
	}

	s := pulumi.ToSecret(pulumi.StringPtr(string(b))).(pulumi.StringPtrOutput)

	return s, nil
}

func WriteStringCommand(filePath string) pulumi.StringInput {
	return pulumi.Sprintf(`cat - > %s`, filePath)
}
