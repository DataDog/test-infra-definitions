package utils

import (
	"encoding/base64"
	"encoding/json"

	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"gopkg.in/yaml.v3"

	"github.com/DataDog/test-infra-definitions/common/config"
)

const imagePullSecretName = "registry-credentials"

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

// NewImagePullSecret creates an image pull secret based on environment
func NewImagePullSecret(e config.Env, namespace string, opts ...pulumi.ResourceOption) (*corev1.Secret, error) {
	dockerConfigJSON := e.ImagePullPassword().ApplyT(func(password string) (string, error) {
		dockerConfigJSON, err := json.Marshal(map[string]map[string]map[string]string{
			"auths": {
				e.ImagePullRegistry(): {
					"username": e.ImagePullUsername(),
					"password": password,
					"auth":     base64.StdEncoding.EncodeToString([]byte(e.ImagePullUsername() + ":" + password)),
				},
			},
		})
		return string(dockerConfigJSON), err
	}).(pulumi.StringOutput)

	return corev1.NewSecret(
		e.Ctx(),
		imagePullSecretName,
		&corev1.SecretArgs{
			Metadata: metav1.ObjectMetaArgs{
				Namespace: pulumi.StringPtr(namespace),
				Name:      pulumi.StringPtr(imagePullSecretName),
			},
			StringData: pulumi.StringMap{
				".dockerconfigjson": dockerConfigJSON,
			},
			Type: pulumi.StringPtr("kubernetes.io/dockerconfigjson"),
		},
		opts...,
	)
}
