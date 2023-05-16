package config

import (
	"strings"

	"github.com/Masterminds/semver"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
)

func FindEnvironmentName(environments []string, prefix string) string {
	for _, env := range environments {
		if strings.HasPrefix(env, prefix+"/") {
			return env
		}
	}
	return ""
}

func SetConfigDefaultValue(config auto.ConfigMap, key, value string) {
	if _, found := config[key]; !found {
		config[key] = auto.ConfigValue{
			Value: value,
		}
	}
}

func anyAgentSemverVersion(version string) (*semver.Version, error) {
	if version == "" {
		return nil, nil
	}

	return semver.NewVersion(version)
}

func AgentSemverVersion(e *CommonEnvironment) (*semver.Version, error) {
	return anyAgentSemverVersion(e.AgentVersion())
}

func ClusterAgentSemverVersion(e *CommonEnvironment) (*semver.Version, error) {
	return anyAgentSemverVersion(e.ClusterAgentVersion())
}
