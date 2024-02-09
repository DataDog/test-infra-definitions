package agent

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func EKSFargateContainerDefinition(e config.CommonEnvironment, image string, clusterName string, apiKeySSMParamName pulumi.StringInput, fakeintake *fakeintake.Fakeintake) *corev1.ContainerArgs {
	if image == "" {
		image = dockerAgentFullImagePath(&e, "public.ecr.aws/datadog/agent", "latest")
	}

	return &corev1.ContainerArgs{
		Name:  pulumi.String("datadog-agent"),
		Image: pulumi.String(image),
		Resources: &corev1.ResourceRequirementsArgs{
			Limits: pulumi.StringMap{
				"cpu":    pulumi.String("100m"),
				"memory": pulumi.String("320Mi"),
			},
			Requests: pulumi.StringMap{
				"cpu":    pulumi.String("10m"),
				"memory": pulumi.String("320Mi"),
			},
		},
		Env: append(corev1.EnvVarArray{
			&corev1.EnvVarArgs{
				Name:  pulumi.String("DD_API_KEY"),
				Value: apiKeySSMParamName,
			},
			&corev1.EnvVarArgs{
				Name:  pulumi.String("DD_SITE"),
				Value: pulumi.String("datadoghq.com"),
			},
			&corev1.EnvVarArgs{
				Name:  pulumi.String("DD_DOGSTATSD_SOCKET"),
				Value: pulumi.String("/var/run/datadog/dsd.socket"),
			},
			&corev1.EnvVarArgs{
				Name:  pulumi.String("DD_CHECKS_TAG_CARDINALITY"),
				Value: pulumi.String("high"),
			},
			&corev1.EnvVarArgs{
				Name:  pulumi.String("DD_EKS_FARGATE"),
				Value: pulumi.String("true"),
			},
			&corev1.EnvVarArgs{
				Name:  pulumi.String("DD_ORCHESTRATOR_EXPLORER_ENABLED"),
				Value: pulumi.String("true"),
			},
			&corev1.EnvVarArgs{
				Name:  pulumi.String("DD_CLUSTER_NAME"),
				Value: pulumi.String(clusterName),
			},
		}, eksFakeintakeAdditionalEndpointsEnv(fakeintake)...),
		Ports: &corev1.ContainerPortArray{
			&corev1.ContainerPortArgs{
				Name:          pulumi.String("udp"),
				ContainerPort: pulumi.Int(8125),
				Protocol:      pulumi.String("UDP"),
			},
			&corev1.ContainerPortArgs{
				Name:          pulumi.String("tcp"),
				ContainerPort: pulumi.Int(8126),
				Protocol:      pulumi.String("TCP"),
			},
		},
	}
}

func eksFakeintakeAdditionalEndpointsEnv(fakeintake *fakeintake.Fakeintake) corev1.EnvVarArray {
	if fakeintake == nil {
		return corev1.EnvVarArray{}
	}
	return corev1.EnvVarArray{
		&corev1.EnvVarArgs{
			Name:  pulumi.String("DD_SKIP_SSL_VALIDATION"),
			Value: pulumi.String("true"),
		},
		&corev1.EnvVarArgs{
			Name:  pulumi.String("DD_REMOTE_CONFIGURATION_NO_TLS_VALIDATION"),
			Value: pulumi.String("true"),
		},
		&corev1.EnvVarArgs{
			Name:  pulumi.String("DD_ADDITIONAL_ENDPOINTS"),
			Value: pulumi.Sprintf(`{"https://%s": ["FAKEAPIKEY"]}`, fakeintake.Host),
		},
		&corev1.EnvVarArgs{
			Name:  pulumi.String("DD_LOGS_CONFIG_ADDITIONAL_ENDPOINTS"),
			Value: pulumi.Sprintf(`[{"host": "%s"}]`, fakeintake.Host),
		},
		&corev1.EnvVarArgs{
			Name:  pulumi.String("DD_LOGS_CONFIG_USE_HTTP"),
			Value: pulumi.String("true"),
		},
	}
}
