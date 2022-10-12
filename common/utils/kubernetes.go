package utils

import (
	"encoding/json"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func KubeconfigToJSON(kubeConfig pulumi.Output) pulumi.StringInput {
	return kubeConfig.ApplyT(func(config interface{}) (string, error) {
		jsonConfig, err := json.Marshal(config)
		if err != nil {
			return "", err
		}

		return string(jsonConfig), nil
	}).(pulumi.StringInput)
}
