package aws

import (
	ddfakeintake "github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
	resourcesAws "github.com/DataDog/test-infra-definitions/resources/aws"
)

func NewEcsFakeintake(env resourcesAws.Environment) (*ddfakeintake.ConnectionExporter, error) {
	comp, err := ddfakeintake.FargateLinuxComponentDefinition(env)
	if err != nil {
		return nil, err
	}

	return ddfakeintake.NewExporter(env.Ctx, comp.Host), nil
}
