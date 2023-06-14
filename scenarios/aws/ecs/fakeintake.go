package ecs

import (
	ddfakeintake "github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/resources/aws/ecs"
)

// NewEcsFakeintake creates a new instance of fakeintake service on a dedicated fargate cluster
// and registers it into the pulumi context
func NewEcsFakeintake(env aws.Environment) (*ddfakeintake.ConnectionExporter, error) {
	ipAddress, err := ecs.FargateServiceFakeintake(env)
	if err != nil {
		return nil, err
	}

	return ddfakeintake.NewExporter(env.Ctx, ipAddress), nil
}
