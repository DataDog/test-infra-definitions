package registry

import (
	"strings"

	dockervm "github.com/DataDog/test-infra-definitions/scenarios/aws/dockerVM"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/ecs"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/eks"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/kindvm"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/microVMs/microvms"
	awsvm "github.com/DataDog/test-infra-definitions/scenarios/aws/vm"
	"github.com/DataDog/test-infra-definitions/scenarios/azure/aks"
	azvm "github.com/DataDog/test-infra-definitions/scenarios/azure/vm"

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
		"aws/kind":     kindvm.Run,
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
