package common

import (
	"github.com/DataDog/test-infra-definitions/common/config"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Run(ctx *pulumi.Context, runFunc func(*pulumi.Context, config.Environment) error) error {
	env, err := GetEnvironmentFromConfig(ctx)
	if err != nil {
		return err
	}

	return runFunc(ctx, env)
}
