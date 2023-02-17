package utils

import (
	"os"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func ReadSecretFile(filePath string) (pulumi.StringOutput, error) {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return pulumi.StringOutput{}, err
	}

	s := pulumi.ToSecret(pulumi.String(string(b))).(pulumi.StringOutput)

	return s, nil
}

func WriteStringCommand(filePath string, useSudo bool) pulumi.StringInput {
	return writeStringCommand(filePath, useSudo, "")
}

func AppendStringCommand(filePath string, useSudo bool) pulumi.StringInput {
	return writeStringCommand(filePath, useSudo, "-a")
}

func writeStringCommand(filePath string, useSudo bool, teeFlags string) pulumi.StringInput {
	sudo := ""
	if useSudo {
		sudo = "sudo"
	}
	return pulumi.Sprintf(`cat - | %s tee %s %s > /dev/null`, sudo, teeFlags, filePath)
}
