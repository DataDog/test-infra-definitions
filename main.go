package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/DataDog/test-infra-definitions/registry"
	_ "github.com/DataDog/test-infra-definitions/registry/scenarios"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

const (
	scenarioEnvVarName = "PULUMI_SCENARIO"
	scenarioParamName  = "scenario"

	dummyScenario = "dummy"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		scenarioName := os.Getenv(scenarioEnvVarName)
		rootConfig := config.New(ctx, "")
		if s := rootConfig.Get(scenarioParamName); s != "" {
			scenarioName = s
		}

		if scenarioName == dummyScenario {
			return nil
		}

		rf := registry.Scenarios.Get(scenarioName)
		if rf == nil {
			return fmt.Errorf("impossible to run unknown scenario: %s, known scenarios: %s", scenarioName, strings.Join(registry.Scenarios.List(), " ,"))
		}

		return rf(ctx)
	})
}
