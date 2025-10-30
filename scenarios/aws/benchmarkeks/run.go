package benchmarkeks

import (
	_ "embed"

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

//go:embed simple_check.py
var simpleCheckPy string

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
				"benchmark.datadoghq.com/agent":   pulumi.String(ng.agent),
				"benchmark.datadoghq.com/variant": pulumi.String(ng.variant),
			},
			Taints: awsEks.NodeGroupTaintArray{
				awsEks.NodeGroupTaintArgs{
					Key:    pulumi.String("benchmark.datadoghq.com/agent"),
					Value:  pulumi.String(ng.agent),
					Effect: pulumi.String("NO_SCHEDULE"),
				},
				awsEks.NodeGroupTaintArgs{
					Key:    pulumi.String("benchmark.datadoghq.com/variant"),
					Value:  pulumi.String(ng.variant),
					Effect: pulumi.String("NO_SCHEDULE"),
				},
			},
			RemoteAccess: awsEks.NodeGroupRemoteAccessArgs{
				Ec2SshKey:              pulumi.StringPtr(awsEnv.DefaultKeyPairName()),
				SourceSecurityGroupIds: pulumi.ToStringArray(awsEnv.EKSAllowedInboundSecurityGroups()),
			},
		}, awsEnv.WithProviders(config.ProviderAWS, config.ProviderEKS)); err != nil {
			return err
		}
	}

	if _, err := kwok.K8sAppDefinition(&awsEnv, cluster.KubeProvider); err != nil {
		return err
	}

	for _, param := range []struct {
		variant               string
		agentImagePath        string
		clusterAgentImagePath string
		deployCRDs            bool
	}{
		{
			variant:               "baseline",
			agentImagePath:        awsEnv.AgentBaselineFullImagePath(),
			clusterAgentImagePath: awsEnv.ClusterAgentBaselineFullImagePath(),
			deployCRDs:            true,
		},
		{
			variant:               "comparison",
			agentImagePath:        awsEnv.AgentComparisonFullImagePath(),
			clusterAgentImagePath: awsEnv.ClusterAgentComparisonFullImagePath(),
			deployCRDs:            false,
		},
	} {
		if _, err := helm.NewKubernetesAgent(&awsEnv, "dda-"+param.variant, awsEnv.Namer.ResourceName("datadog-agent", param.variant), cluster.KubeProvider,
			kubernetesagentparams.WithNamespace("datadog-"+param.variant),
			kubernetesagentparams.WithClusterName(cluster.ClusterName),
			kubernetesagentparams.WithAgentFullImagePath(param.agentImagePath),
			kubernetesagentparams.WithClusterAgentFullImagePath(param.clusterAgentImagePath),
			kubernetesagentparams.WithHelmValues(utils.YAMLMustMarshal(map[string]any{
				"datadog": map[string]any{
					"nodeLabelsAsTags": map[string]any{
						"benchmark.datadoghq.com/agent":   "agent",
						"benchmark.datadoghq.com/variant": "variant",
					},
					"podLabelsAsTags": map[string]any{
						"app": "app",
					},
					"env": []map[string]string{
						{
							"name":  "DD_INTERNAL_PROFILING_BLOCK_PROFILE_RATE",
							"value": "10000",
						},
						{
							"name":  "DD_INTERNAL_PROFILING_ENABLE_BLOCK_PROFILING",
							"value": "true",
						},
						{
							"name":  "DD_INTERNAL_PROFILING_ENABLE_GOROUTINE_STACKTRACES",
							"value": "true",
						},
						{
							"name":  "DD_INTERNAL_PROFILING_ENABLE_MUTEX_PROFILING",
							"value": "true",
						},
						{
							"name":  "DD_INTERNAL_PROFILING_ENABLED",
							"value": "true",
						},
						{
							"name":  "DD_INTERNAL_PROFILING_MUTEX_PROFILE_FRACTION",
							"value": "100",
						},
					},
					"checksd": map[string]string{
						"simple_check.py": simpleCheckPy,
					},
				},
				"agents": map[string]any{
					"nodeSelector": map[string]any{
						"benchmark.datadoghq.com/variant": param.variant,
					},
					"tolerations": []any{
						map[string]any{
							"key":      "benchmark.datadoghq.com/agent",
							"operator": "Exists",
							"effect":   "NoSchedule",
						},
						map[string]any{
							"key":      "benchmark.datadoghq.com/variant",
							"operator": "Equal",
							"value":    param.variant,
							"effect":   "NoSchedule",
						},
					},
				},
				"clusterAgent": map[string]any{
					"replicas": 1,
					"nodeSelector": map[string]any{
						"benchmark.datadoghq.com/agent":   "cluster-agent",
						"benchmark.datadoghq.com/variant": param.variant,
					},
					"tolerations": []any{
						map[string]any{
							"key":      "benchmark.datadoghq.com/agent",
							"operator": "Equal",
							"value":    "cluster-agent",
							"effect":   "NoSchedule",
						},
						map[string]any{
							"key":      "benchmark.datadoghq.com/variant",
							"operator": "Equal",
							"value":    param.variant,
							"effect":   "NoSchedule",
						},
					},
					"metricsProvider": map[string]any{
						"registerAPIService": false,
					},
					"admissionController": map[string]any{
						"agentSidecarInjection": map[string]any{
							"clusterAgentCommunicationEnabled": false,
						},
					},
				},
				"clusterChecksRunner": map[string]any{
					"replicas": 1,
					"nodeSelector": map[string]any{
						"benchmark.datadoghq.com/agent":   "cluster-checks",
						"benchmark.datadoghq.com/variant": param.variant,
					},
					"tolerations": []any{
						map[string]any{
							"key":      "benchmark.datadoghq.com/agent",
							"operator": "Equal",
							"value":    "cluster-checks",
							"effect":   "NoSchedule",
						},
						map[string]any{
							"key":      "benchmark.datadoghq.com/variant",
							"operator": "Equal",
							"value":    param.variant,
							"effect":   "NoSchedule",
						},
					},
				},
				"datadog-crds": map[string]any{
					"crds": map[string]any{
						"datadogMetrics":        param.deployCRDs,
						"datadogPodAutoscalers": param.deployCRDs,
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
