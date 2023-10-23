package aws

import (
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/fakeintake"
	ddfakeintake "github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
	resourcesAws "github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/fakeintake/fakeintakeparams"
)

func NewEcsFakeintake(env resourcesAws.Environment, options ...fakeintakeparams.Option) (*ddfakeintake.ConnectionExporter, error) {
	fargateInstance, err := fakeintake.NewECSFargateInstance(env, options...)
	if err != nil {
		return nil, err
	}

	return ddfakeintake.NewExporter(env.Ctx, fargateInstance.Host, fargateInstance.Name), nil
}
