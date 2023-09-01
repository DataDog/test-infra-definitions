package aws

import (
	ddfakeintake "github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
	resourcesAws "github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/resources/aws/ecs"
)

func NewEcsFakeintake(env resourcesAws.Environment) (*ddfakeintake.ConnectionExporter, error) {
	fakeintake, err := ddfakeintake.NewECSFargateInstance(env, ecs.OldFargate{})
	if err != nil {
		return nil, err
	}

	return ddfakeintake.NewExporter(env.Ctx, fakeintake.Host), nil
}
