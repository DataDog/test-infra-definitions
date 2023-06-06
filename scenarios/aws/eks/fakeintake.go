package eks

import (
	ddfakeintake "github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/resources/aws/ecs"
)

func NewEcsFakeintake(env aws.Environment) (*ddfakeintake.ConnectionExporter, error) {
	ipAddress, err := ecs.FargateServiceFakeintake(env)
	if err != nil {
		return nil, err
	}

	return ddfakeintake.NewExporter(env.Ctx, ipAddress), nil
}
