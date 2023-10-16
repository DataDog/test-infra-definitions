package aws

import (
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/fakeintake"
	ddfakeintake "github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
	resourcesAws "github.com/DataDog/test-infra-definitions/resources/aws"
)

func NewEcsFakeintake(env resourcesAws.Environment) (*ddfakeintake.ConnectionExporter, error) {
	return NewEcsFakeintakeWithName(env, "fakeintake")
}

func NewEcsFakeintakeWithName(env resourcesAws.Environment, name string) (*ddfakeintake.ConnectionExporter, error) {
	fargateInstance, err := fakeintake.NewECSFargateInstance(env, name)
	if err != nil {
		return nil, err
	}

	return ddfakeintake.NewExporter(env.Ctx, fargateInstance.Host, name), nil
}
