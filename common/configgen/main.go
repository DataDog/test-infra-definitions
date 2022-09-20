package main

import (
	"os"

	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/common/config"

	"gopkg.in/yaml.v3"
)

func GenerateConfigFile(envConfig config.EnvironmentConfig, filePath string) error {
	data, err := yaml.Marshal(envConfig)
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0o660)
}

func main() {
	// Expects to be run from Git root
	GenerateConfigFile(aws.GetSandboxEnvironmentConfig(), "./envConfigs/aws-sandbox.yaml")
}
