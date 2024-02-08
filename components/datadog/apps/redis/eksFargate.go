package redis

import (
	"github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/eks"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type EksFargateComponent struct {
	pulumi.ResourceState
}

func EKSFargateAppDefintion(e aws.Environment, kubeProvider *kubernetes.Provider, namespace string, clusterName string, opts ...pulumi.ResourceOption) (*EksFargateComponent, error) {
	// opts = append(opts, e.WithProviders(config.ProviderAWS, config.ProviderAWSX))

	eksFargateComponent := &EksFargateComponent{}
	if err := e.Ctx.RegisterComponentResource("dd:apps", "redis", eksFargateComponent); err != nil {
		return nil, err
	}

	// kubeVersion, err := semver.NewVersion(e.KubernetesVersion())
	// if err != nil {
	// 	return nil, err
	// }

	// opts = append(opts, pulumi.Parent(eksFargateComponent))
	// opts = append(opts, utils.PulumiDependsOn(ns))

	// _, err := eks.NewFargateProfile(e.Ctx, "redis", &eks.FargateProfileArgs{
	// 	ClusterName:         pulumi.String(clusterName),
	// 	PodExecutionRoleArn: pulumi.String("redis"),
	// 	FargateProfileName:  pulumi.String("redis"),
	// 	Selectors: eks.FargateProfileSelectorArray{
	// 		&eks.FargateProfileSelectorArgs{
	// 			Namespace: pulumi.String("example"),
	// 		},
	// 	},
	// })
	_, err := eks.NewFargateProfile(e.Ctx, "redis", &eks.FargateProfileArgs{
		ClusterName: pulumi.String(clusterName),
		Selectors: eks.FargateProfileSelectorArray{
			&eks.FargateProfileSelectorArgs{
				Namespace: pulumi.String(namespace),
			},
		},
	})

	// fargateProfile := pulumi.Any(eks.FargateProfile{
	// 	eks.FargateProfileArgs{
	// 		ClusterName: pulumi.Any(clusterName),
	// 	},
	// 	Selectors: []awsEks.FargateProfileSelector{
	// 		{
	// 			Namespace: namespace,
	// 		},
	// 	},
	// })
	// 	PodExecutionRoleArn: pulumi.String("redis"),
	// 	FargateProfileName:  pulumi.String("redis"),
	// 	Selectors: []awsEks.FargateProfileSelector{
	// 		{
	// 			Namespace: fargateNamespace,
	// 		},
	// 	},
	// }

	// _, err := eks.NewFargateProfile(ctx, "example", &eks.FargateProfileArgs{
	// 	ClusterName:         pulumi.Any(aws_eks_cluster.Example.Name),
	// 	PodExecutionRoleArn: pulumi.Any(aws_iam_role.Example.Arn),
	// 	SubnetIds:           toPulumiArray(splat0),
	// 	Selectors: eks.FargateProfileSelectorArray{
	// 		&eks.FargateProfileSelectorArgs{
	// 			Namespace: pulumi.String("example"),
	// 		},
	// 	},
	// })

	return eksFargateComponent, err
}
