package ecs

import (
	ddfakeintake "github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
	"github.com/DataDog/test-infra-definitions/resources/aws/ecs"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/vm/ec2vm"
)

// NewEcsFakeintake creates a new instance of fakeintake service on a dedicated fargate cluster
// and registers it into the pulumi context
func NewEcsFakeintake(infra ec2vm.Infra) (exporter *ddfakeintake.ConnectionExporter, err error) {
	ipAddress, err := ecs.FargateServiceFakeintake(infra.GetAwsEnvironment())
	if err != nil {
		return nil, err
	}

	exporter = ddfakeintake.NewExporter(infra.GetAwsEnvironment().Ctx, ipAddress)

	return
}
