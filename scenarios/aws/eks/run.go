package eks

import (
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/datadog/agent"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/dogstatsd"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/nginx"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/prometheus"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/redis"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	localEks "github.com/DataDog/test-infra-definitions/resources/aws/eks"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	awsEks "github.com/pulumi/pulumi-aws/sdk/v5/go/aws/eks"
	awsIam "github.com/pulumi/pulumi-aws/sdk/v5/go/aws/iam"
	"github.com/pulumi/pulumi-eks/sdk/go/eks"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Run(ctx *pulumi.Context) error {
	awsEnv, err := aws.NewEnvironment(ctx)
	if err != nil {
		return err
	}

	// Create Cluster SG
	clusterSG, err := ec2.NewSecurityGroup(ctx, awsEnv.Namer.ResourceName("eks-sg"), &ec2.SecurityGroupArgs{
		NamePrefix:  awsEnv.CommonNamer.DisplayName(pulumi.String("eks-sg")),
		Description: pulumi.StringPtr("EKS Cluster sg for stack: " + ctx.Stack()),
		Ingress: ec2.SecurityGroupIngressArray{
			ec2.SecurityGroupIngressArgs{
				SecurityGroups: pulumi.ToStringArray(awsEnv.EKSAllowedInboundSecurityGroups()),
				ToPort:         pulumi.Int(22),
				FromPort:       pulumi.Int(22),
				Protocol:       pulumi.String("tcp"),
			},
			ec2.SecurityGroupIngressArgs{
				SecurityGroups: pulumi.ToStringArray(awsEnv.EKSAllowedInboundSecurityGroups()),
				ToPort:         pulumi.Int(443),
				FromPort:       pulumi.Int(443),
				Protocol:       pulumi.String("tcp"),
			},
		},
		VpcId: pulumi.StringPtr(awsEnv.DefaultVPCID()),
	}, awsEnv.ResourceProvidersOption())
	if err != nil {
		return err
	}

	// Cluster role
	clusterRole, err := localEks.GetClusterRole(awsEnv, "eks-cluster-role")
	if err != nil {
		return err
	}

	// IAM Node role
	linuxNodeRole, err := localEks.GetNodeRole(awsEnv, "eks-linux-node-role")
	if err != nil {
		return err
	}

	windowsNodeRole, err := localEks.GetNodeRole(awsEnv, "eks-windows-node-role")
	if err != nil {
		return err
	}

	// Fargate Configuration
	var fargateProfile pulumi.Input
	if fargateNamespace := awsEnv.EKSFargateNamespace(); fargateNamespace != "" {
		fargateProfile = pulumi.Any(
			eks.FargateProfile{
				Selectors: []awsEks.FargateProfileSelector{
					{
						Namespace: fargateNamespace,
					},
				},
			},
		)
	}

	// Create an EKS cluster with the default configuration.
	cluster, err := eks.NewCluster(ctx, awsEnv.Namer.ResourceName("eks"), &eks.ClusterArgs{
		Name:                         awsEnv.CommonNamer.DisplayName(),
		Version:                      pulumi.StringPtr(awsEnv.KubernetesVersion()),
		EndpointPrivateAccess:        pulumi.BoolPtr(true),
		EndpointPublicAccess:         pulumi.BoolPtr(false),
		Fargate:                      fargateProfile,
		ClusterSecurityGroup:         clusterSG,
		NodeAssociatePublicIpAddress: pulumi.BoolRef(false),
		PrivateSubnetIds:             pulumi.ToStringArray(awsEnv.DefaultSubnets()),
		VpcId:                        pulumi.StringPtr(awsEnv.DefaultVPCID()),
		SkipDefaultNodeGroup:         pulumi.BoolRef(true),
		// The content of the aws-auth map is the merge of `InstanceRoles` and `RoleMappings`.
		// For managed node groups, we push the value in `InstanceRoles`.
		// For unmanaged node groups, we push the value in `RoleMappings`
		RoleMappings: eks.RoleMappingArray{
			eks.RoleMappingArgs{
				Groups:   pulumi.ToStringArray([]string{"system:bootstrappers", "system:nodes", "eks:kube-proxy-windows"}),
				Username: pulumi.String("system:node:{{EC2PrivateDNSName}}"),
				RoleArn:  windowsNodeRole.Arn,
			},
		},
		InstanceRoles: awsIam.RoleArray{
			linuxNodeRole,
		},
		ServiceRole: clusterRole,
	}, pulumi.Timeouts(&pulumi.CustomTimeouts{
		Create: "30m",
		Update: "30m",
		Delete: "30m",
	}))
	if err != nil {
		return err
	}

	nodeGroups := make([]pulumi.Resource, 0)
	// Create managed node groups
	if awsEnv.EKSLinuxNodeGroup() {
		ng, err := localEks.NewLinuxNodeGroup(awsEnv, cluster, linuxNodeRole)
		if err != nil {
			return err
		}
		nodeGroups = append(nodeGroups, ng)
	}

	if awsEnv.EKSLinuxARMNodeGroup() {
		ng, err := localEks.NewLinuxARMNodeGroup(awsEnv, cluster, linuxNodeRole)
		if err != nil {
			return err
		}
		nodeGroups = append(nodeGroups, ng)
	}

	if awsEnv.EKSBottlerocketNodeGroup() {
		ng, err := localEks.NewBottlerocketNodeGroup(awsEnv, cluster, linuxNodeRole)
		if err != nil {
			return err
		}
		nodeGroups = append(nodeGroups, ng)
	}

	// Create unmanaged node groups
	if awsEnv.EKSWindowsNodeGroup() {
		_, err := localEks.NewWindowsUnmanagedNodeGroup(awsEnv, cluster, windowsNodeRole)
		if err != nil {
			return err
		}
	}

	// Export the cluster's kubeconfig.
	ctx.Export("kubeconfig", cluster.Kubeconfig)

	// Building Kubernetes provider
	eksKubeProvider, err := kubernetes.NewProvider(awsEnv.Ctx, awsEnv.Namer.ResourceName("k8s-provider"), &kubernetes.ProviderArgs{
		EnableServerSideApply: pulumi.BoolPtr(true),
		Kubeconfig:            utils.KubeconfigToJSON(cluster.Kubeconfig),
	}, awsEnv.ResourceProvidersOption(), pulumi.DependsOn(nodeGroups))
	if err != nil {
		return err
	}

	// Applying necessary Windows configuration if Windows nodes
	if awsEnv.EKSWindowsNodeGroup() {
		_, err := corev1.NewConfigMapPatch(awsEnv.Ctx, awsEnv.Namer.ResourceName("eks-cni-cm"), &corev1.ConfigMapPatchArgs{
			Metadata: metav1.ObjectMetaPatchArgs{
				Namespace: pulumi.String("kube-system"),
				Name:      pulumi.String("amazon-vpc-cni"),
			},
			Data: pulumi.StringMap{
				"enable-windows-ipam": pulumi.String("true"),
			},
		}, pulumi.Provider(eksKubeProvider))
		if err != nil {
			return err
		}
	}

	// Deploy the Agent
	if awsEnv.AgentDeploy() {
		helmComponent, err := agent.NewHelmInstallation(*awsEnv.CommonEnvironment, agent.HelmInstallationArgs{
			KubeProvider:  eksKubeProvider,
			Namespace:     "datadog",
			DeployWindows: awsEnv.EKSWindowsNodeGroup(),
		}, nil)
		if err != nil {
			return err
		}

		ctx.Export("agent-linux-helm-install-name", helmComponent.LinuxHelmReleaseName)
		ctx.Export("agent-linux-helm-install-status", helmComponent.LinuxHelmReleaseStatus)
		if awsEnv.EKSWindowsNodeGroup() {
			ctx.Export("agent-windows-helm-install-name", helmComponent.WindowsHelmReleaseName)
			ctx.Export("agent-windows-helm-install-status", helmComponent.WindowsHelmReleaseStatus)
		}
	}

	// Deploy testing workload
	if awsEnv.TestingWorkloadDeploy() {
		if _, err := nginx.K8sAppDefinition(*awsEnv.CommonEnvironment, eksKubeProvider, "workload-nginx"); err != nil {
			return err
		}

		if _, err := redis.K8sAppDefinition(*awsEnv.CommonEnvironment, eksKubeProvider, "workload-redis"); err != nil {
			return err
		}

		if _, err := dogstatsd.K8sAppDefinition(*awsEnv.CommonEnvironment, eksKubeProvider, "workload-dogstatsd"); err != nil {
			return err
		}

		if _, err := prometheus.K8sAppDefinition(*awsEnv.CommonEnvironment, eksKubeProvider, "workload-prometheus"); err != nil {
			return err
		}
	}

	return nil
}
