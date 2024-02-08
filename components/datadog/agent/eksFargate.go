package agent

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"

	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/ecs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func EKSFargateLinuxContainerDefinition(e config.CommonEnvironment, image string, apiKeySSMParamName pulumi.StringInput, fakeintake *fakeintake.Fakeintake, logConfig ecs.TaskDefinitionLogConfigurationPtrInput) *corev1.ContainerArgs {
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
		Env: &corev1.EnvVarArray{
			&corev1.EnvVarArgs{
				Name:  pulumi.String("DD_API_KEY"),
				Value: apiKeySSMParamName,
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
		},
		// LivenessProbe: &corev1.ProbeArgs{
		// 	HttpGet: &corev1.HTTPGetActionArgs{
		// 		Port: pulumi.Int(80),
		// 	},
		// },
		// ReadinessProbe: &corev1.ProbeArgs{
		// 	HttpGet: &corev1.HTTPGetActionArgs{
		// 		Port: pulumi.Int(80),
		// 	},
		// },
	}
}
