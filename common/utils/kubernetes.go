package utils

import (
	"encoding/json"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"gopkg.in/yaml.v3"
)

// KubeConfigYAMLToJSON safely converts a yaml kubeconfig to a json string.
func KubeConfigYAMLToJSON(kubeConfig pulumi.StringOutput) pulumi.StringInput {
	return kubeConfig.ApplyT(func(config string) (string, error) {
		var body map[string]interface{}
		err := yaml.Unmarshal([]byte(config), &body)
		if err != nil {
			return "", err
		}

		jsonConfig, err := json.Marshal(body)
		if err != nil {
			return "", err
		}
		return string(jsonConfig), nil
	}).(pulumi.StringInput)
}
