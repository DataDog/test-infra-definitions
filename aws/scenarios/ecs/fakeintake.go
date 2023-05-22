package ecs

import (
	"github.com/DataDog/test-infra-definitions/aws/ecs"
	ec2vm "github.com/DataDog/test-infra-definitions/aws/scenarios/vm/ec2VM"
	ddfakeintake "github.com/DataDog/test-infra-definitions/datadog/fakeintake"
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
