package config

import (
	"github.com/Masterminds/semver"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
)

func SetConfigDefaultValue(config auto.ConfigMap, key, value string) {
	if _, found := config[key]; !found {
		config[key] = auto.ConfigValue{
			Value: value,
		}
	}
}

func AgentSemverVersion(e CommonEnvironment) (*semver.Version, error) {
	version := e.AgentVersion()
	if version == "" {
		return nil, nil
	}

	return semver.NewVersion(version)
}
