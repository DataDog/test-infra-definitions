package scenarios

import (
	// Centralize import of all scenarios
	_ "github.com/DataDog/test-infra-definitions/aws/scenarios/dockerVM"
	_ "github.com/DataDog/test-infra-definitions/aws/scenarios/ecs"
	_ "github.com/DataDog/test-infra-definitions/aws/scenarios/eks"
	_ "github.com/DataDog/test-infra-definitions/aws/scenarios/microVMs/microvms"
)
