package registry

import (
	"strings"

	dockervm "github.com/DataDog/test-infra-definitions/aws/scenarios/dockerVM"
	"github.com/DataDog/test-infra-definitions/aws/scenarios/ecs"
	"github.com/DataDog/test-infra-definitions/aws/scenarios/eks"
	"github.com/DataDog/test-infra-definitions/aws/scenarios/microVMs/microvms"
	awsvm "github.com/DataDog/test-infra-definitions/aws/scenarios/vm"
	"github.com/DataDog/test-infra-definitions/azure/scenarios/aks"
	azvm "github.com/DataDog/test-infra-definitions/azure/scenarios/vm"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type ScenarioRegistry map[string]pulumi.RunFunc

func Scenarios() ScenarioRegistry {
	return ScenarioRegistry{
		"aws/vm":       awsvm.Run,
		"aws/dockervm": dockervm.Run,
		"aws/ecs":      ecs.Run,
		"aws/eks":      eks.Run,
		"aws/microvms": microvms.Run,
		"az/vm":        azvm.Run,
		"az/aks":       aks.Run,
	}
}

func (s ScenarioRegistry) Get(name string) pulumi.RunFunc {
	return s[strings.ToLower(name)]
}

func (s ScenarioRegistry) List() []string {
	names := make([]string, 0, len(s))
	for name := range s {
		names = append(names, name)
	}

	return names
}
