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

func AgentSemverVersion(e *CommonEnvironment) (*semver.Version, error) {
	version := e.AgentVersion()
	if version == "" {
		return nil, nil
	}

	return semver.NewVersion(version)
}
