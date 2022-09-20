package config

import "github.com/pulumi/pulumi/sdk/v3/go/auto"

func SetConfigDefaultValue(config auto.ConfigMap, key, value string) {
	if _, found := config[key]; !found {
		config[key] = auto.ConfigValue{
			Value: value,
		}
	}
}
