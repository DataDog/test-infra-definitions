package common

import (
	"errors"
	"strings"

	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/common/config"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	sdkconfig "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func GetEnvironmentFromConfig(ctx *pulumi.Context) (config.Environment, error) {
	environment := strings.ToLower(sdkconfig.Require(ctx, config.GetParamKey("environment")))
	switch environment {
	case "aws-sandbox":
		return aws.SandboxEnvironment{}, nil
	default:
		return nil, errors.New("unknown envrionment")
	}
}
