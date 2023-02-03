package os

import (
	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ssm"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func GetLatestAMI(e aws.Environment, arch Architecture, amd64Path string, armPath string) (string, error) {
	amiParamName := amd64Path
	if arch == ARM64Arch {
		amiParamName = armPath
	}
	result, err := ssm.LookupParameter(e.Ctx, &ssm.LookupParameterArgs{
		Name: amiParamName,
	}, pulumi.Provider(e.Provider))
	if err != nil {
		return "", err
	}
	return result.Value, nil
}
