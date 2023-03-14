package registry

import (
	"strings"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type scenarioRegistry map[string]pulumi.RunFunc

var Scenarios = make(scenarioRegistry)

// Register lower case the given name
func (s scenarioRegistry) Register(name string, rf pulumi.RunFunc) {
	s[strings.ToLower(name)] = rf
}

func (s scenarioRegistry) Get(name string) pulumi.RunFunc {
	return s[strings.ToLower(name)]
}

func (s scenarioRegistry) List() []string {
	names := make([]string, 0, len(s))
	for name := range s {
		names = append(names, name)
	}

	return names
}
