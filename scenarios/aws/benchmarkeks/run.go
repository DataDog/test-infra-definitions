package benchmarkeks

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/datadog/agent/helm"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/churn"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/kwok"
	"github.com/DataDog/test-infra-definitions/components/datadog/kubernetesagentparams"
	resourcesAws "github.com/DataDog/test-infra-definitions/resources/aws"
	scenarioEks "github.com/DataDog/test-infra-definitions/scenarios/aws/eks"
	awsEks "github.com/pulumi/pulumi-aws/sdk/v6/go/aws/eks"

	"github.com/pulumi/pulumi-eks/sdk/v3/go/eks"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	instanceType = "t3.medium"
	nbNode       = 3
)

func Run(ctx *pulumi.Context) error {
	awsEnv, err := resourcesAws.NewEnvironment(ctx)
	if err != nil {
		return err
	}

	cluster, err := scenarioEks.NewCluster(awsEnv, "eks")
	if err != nil {
		return err
	}

	if err := cluster.Export(ctx, nil); err != nil {
		return err
	}

	for _, ng := range []struct {
		agent   string
		variant string
		nbNode  int
	}{
		{
			agent:   "node-agent",
			variant: "baseline",
			nbNode:  nbNode,
		},
		{
			agent:   "node-agent",
			variant: "comparison",
			nbNode:  nbNode,
		},
		{
			agent:   "cluster-agent",
			variant: "baseline",
			nbNode:  1,
		},
		{
			agent:   "cluster-agent",
			variant: "comparison",
			nbNode:  1,
		},
		{
			agent:   "cluster-checks",
			variant: "baseline",
			nbNode:  1,
		},
		{
			agent:   "cluster-checks",
			variant: "comparison",
			nbNode:  1,
		},
	} {
		if _, err := eks.NewManagedNodeGroup(ctx, "ng-"+ng.agent+"-"+ng.variant, &eks.ManagedNodeGroupArgs{
			Cluster:             cluster.Cluster.Core,
			InstanceTypes:       pulumi.ToStringArray([]string{instanceType}),
			ForceUpdateVersion:  pulumi.BoolPtr(true),
			NodeGroupNamePrefix: awsEnv.CommonNamer().DisplayName(37, pulumi.String("ng"), pulumi.String(ng.agent), pulumi.String(ng.variant)),
			ScalingConfig: awsEks.NodeGroupScalingConfigArgs{
				DesiredSize: pulumi.Int(ng.nbNode),
				MaxSize:     pulumi.Int(ng.nbNode),
				MinSize:     pulumi.Int(ng.nbNode),
			},
			NodeRole: cluster.Cluster.InstanceRoles.Index(pulumi.Int(0)),
			Labels: pulumi.StringMap{
				"node-role.kubernetes.io/" + ng.agent:   pulumi.String(ng.agent),
				"node-role.kubernetes.io/" + ng.variant: pulumi.String(ng.variant),
			},
			Taints: awsEks.NodeGroupTaintArray{
				awsEks.NodeGroupTaintArgs{
					Key:    pulumi.String("agent"),
					Value:  pulumi.String(ng.agent),
					Effect: pulumi.String("NO_SCHEDULE"),
				},
				awsEks.NodeGroupTaintArgs{
					Key:    pulumi.String("variant"),
					Value:  pulumi.String(ng.variant),
					Effect: pulumi.String("NO_SCHEDULE"),
				},
			},
		}, awsEnv.WithProviders(config.ProviderAWS, config.ProviderEKS)); err != nil {
			return err
		}
	}

	if _, err := kwok.K8sAppDefinition(&awsEnv, cluster.KubeProvider); err != nil {
		return err
	}

	for i, variant := range []string{"baseline", "comparison"} {
		if _, err := helm.NewKubernetesAgent(&awsEnv, "dda-"+variant, awsEnv.Namer.ResourceName("datadog-agent", variant), cluster.KubeProvider,
			kubernetesagentparams.WithNamespace("datadog-"+variant),
			kubernetesagentparams.WithClusterName(cluster.ClusterName),
			kubernetesagentparams.WithHelmValues(utils.YAMLMustMarshal(map[string]any{
				"agents": map[string]any{
					"nodeSelector": map[string]any{
						"node-role.kubernetes.io/" + variant: variant,
					},
					"tolerations": []any{
						map[string]any{
							"key":      "agent",
							"operator": "Exists",
							"effect":   "NoSchedule",
						},
						map[string]any{
							"key":      "variant",
							"operator": "Equal",
							"value":    variant,
							"effect":   "NoSchedule",
						},
					},
				},
				"clusterAgent": map[string]any{
					"replicas": 1,
					"nodeSelector": map[string]any{
						"node-role.kubernetes.io/cluster-agent": "cluster-agent",
						"node-role.kubernetes.io/" + variant:    variant,
					},
					"tolerations": []any{
						map[string]any{
							"key":      "agent",
							"operator": "Equal",
							"value":    "cluster-agent",
							"effect":   "NoSchedule",
						},
						map[string]any{
							"key":      "variant",
							"operator": "Equal",
							"value":    variant,
							"effect":   "NoSchedule",
						},
					},
					"metricsProvider": map[string]any{
						"registerAPIService": false, // i == 0,
					},
				},
				"clusterChecksRunner": map[string]any{
					"replicas": 1,
					"nodeSelector": map[string]any{
						"node-role.kubernetes.io/cluster-checks": "cluster-checks",
						"node-role.kubernetes.io/" + variant:     variant,
					},
					"tolerations": []any{
						map[string]any{
							"key":      "agent",
							"operator": "Equal",
							"value":    "cluster-checks",
							"effect":   "NoSchedule",
						},
						map[string]any{
							"key":      "variant",
							"operator": "Equal",
							"value":    variant,
							"effect":   "NoSchedule",
						},
					},
				},
				"datadog-crds": map[string]any{
					"crds": map[string]any{
						"datadogMetrics":        i == 0,
						"datadogPodAutoscalers": i == 0,
					},
				},
			})),
		); err != nil {
			return err
		}
	}

	if _, err := churn.K8sAppDefinition(&awsEnv, cluster.KubeProvider); err != nil {
		return err
	}

	return nil
}
