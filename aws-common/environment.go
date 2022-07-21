package awscommon

import (
	"errors"
	"strings"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

type Environment interface {
	Region() string
	ECSExecKMSKeyID() string
	VPCID() string
}

func GetEnvironmentFromConfig(ctx *pulumi.Context) (Environment, error) {
	environment := strings.ToLower(config.Require(ctx, "ENVIRONMENT"))
	switch environment {
	case "sandbox":
		return SandboxEnvironment{}, nil
	default:
		return nil, errors.New("Unknown envrionment")
	}
}
