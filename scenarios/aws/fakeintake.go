package aws

import (
	fakeintake "github.com/DataDog/test-infra-definitions/components/datadog/apps/fakeintake"
	ddfakeintake "github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
	resourcesAws "github.com/DataDog/test-infra-definitions/resources/aws"
)

func NewEcsFakeintake(env resourcesAws.Environment) (*ddfakeintake.ConnectionExporter, error) {
	fakeintake, err := fakeintake.NewECSFargateInstance(env)
	if err != nil {
		return nil, err
	}

	return ddfakeintake.NewExporter(env.Ctx, fakeintake.Host), nil
}
