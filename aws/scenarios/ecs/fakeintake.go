package ecs

import (
	"github.com/DataDog/test-infra-definitions/aws/ecs"
	ec2vm "github.com/DataDog/test-infra-definitions/aws/scenarios/vm/ec2VM"
	ddfakeintake "github.com/DataDog/test-infra-definitions/datadog/fakeintake"
)

// NewEcsFakeintake creates a new instance of fakeintake service on a dedicated fargate cluster
// and registers it into the pulumi context
func NewEcsFakeintake(vm *ec2vm.EC2UnixVM) (exporter *ddfakeintake.PulumiExporter, err error) {
	ipAddress, err := ecs.FargateServiceFakeintake(vm.GetAwsEnvironment())
	if err != nil {
		return nil, err
	}

	exporter = ddfakeintake.NewExporter(vm.GetAwsEnvironment().Ctx, ddfakeintake.PulumiData{URL: ipAddress})

	return
}
