package eks

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components"
	"github.com/DataDog/test-infra-definitions/components/datadog/agent"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/cpustress"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/dogstatsd"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/nginx"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/prometheus"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/redis"
	dogstatsdstandalone "github.com/DataDog/test-infra-definitions/components/datadog/dogstatsd-standalone"
	fakeintakeComp "github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
	kubeComp "github.com/DataDog/test-infra-definitions/components/kubernetes"
	resourcesAws "github.com/DataDog/test-infra-definitions/resources/aws"
	localEks "github.com/DataDog/test-infra-definitions/resources/aws/eks"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/fakeintake"

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
	awsEnv, err := resourcesAws.NewEnvironment(ctx)
	if err != nil {
		return err
	}

	clusterComp, err := components.NewComponent(*awsEnv.CommonEnvironment, awsEnv.Namer.ResourceName("eks"), func(comp *kubeComp.Cluster) error {
		// Create Cluster SG
		clusterSG, err := ec2.NewSecurityGroup(ctx, awsEnv.Namer.ResourceName("eks-sg"), &ec2.SecurityGroupArgs{
			NamePrefix:  awsEnv.CommonNamer.DisplayName(255, pulumi.String("eks-sg")),
			Description: pulumi.StringPtr("EKS Cluster sg for stack: " + ctx.Stack()),
			Ingress: ec2.SecurityGroupIngressArray{
				ec2.SecurityGroupIngressArgs{
					SecurityGroups: pulumi.ToStringArray(awsEnv.EKSAllowedInboundSecurityGroups()),
					PrefixListIds:  pulumi.ToStringArray(awsEnv.EKSAllowedInboundPrefixLists()),
					ToPort:         pulumi.Int(22),
					FromPort:       pulumi.Int(22),
					Protocol:       pulumi.String("tcp"),
				},
				ec2.SecurityGroupIngressArgs{
					SecurityGroups: pulumi.ToStringArray(awsEnv.EKSAllowedInboundSecurityGroups()),
					PrefixListIds:  pulumi.ToStringArray(awsEnv.EKSAllowedInboundPrefixLists()),
					ToPort:         pulumi.Int(443),
					FromPort:       pulumi.Int(443),
					Protocol:       pulumi.String("tcp"),
				},
			},
			VpcId: pulumi.StringPtr(awsEnv.DefaultVPCID()),
		}, awsEnv.WithProviders(config.ProviderAWS))
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
			Name:                         awsEnv.CommonNamer.DisplayName(100),
			Version:                      pulumi.StringPtr(awsEnv.KubernetesVersion()),
			EndpointPrivateAccess:        pulumi.BoolPtr(true),
			EndpointPublicAccess:         pulumi.BoolPtr(false),
			Fargate:                      fargateProfile,
			ClusterSecurityGroup:         clusterSG,
			NodeAssociatePublicIpAddress: pulumi.BoolRef(false),
			PrivateSubnetIds:             awsEnv.RandomSubnets(),
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
		}), awsEnv.WithProviders(config.ProviderEKS, config.ProviderAWS))
		if err != nil {
			return err
		}

		// Filling Kubernetes component from EKS cluster
		comp.ClusterName = cluster.EksCluster.Name()
		comp.KubeConfig = cluster.Kubeconfig.AsStringOutput()

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

		// Building Kubernetes provider
		eksKubeProvider, err := kubernetes.NewProvider(awsEnv.Ctx, awsEnv.Namer.ResourceName("k8s-provider"), &kubernetes.ProviderArgs{
			EnableServerSideApply: pulumi.BoolPtr(true),
			Kubeconfig:            utils.KubeconfigToJSON(cluster.Kubeconfig),
		}, awsEnv.WithProviders(config.ProviderAWS), pulumi.DependsOn(nodeGroups))
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

		var dependsOnCrd pulumi.ResourceOption

		var fakeIntake *fakeintakeComp.Fakeintake
		if awsEnv.GetCommonEnvironment().AgentUseFakeintake() {
			if fakeIntake, err = fakeintake.NewECSFargateInstance(awsEnv, "ecs"); err != nil {
				return err
			}
		}

		// Deploy the agent
		if awsEnv.AgentDeploy() {
			helmComponent, err := agent.NewHelmInstallation(*awsEnv.CommonEnvironment, agent.HelmInstallationArgs{
				KubeProvider:  eksKubeProvider,
				Namespace:     "datadog",
				Fakeintake:    fakeIntake,
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

			dependsOnCrd = utils.PulumiDependsOn(helmComponent)
		}

		// Deploy standalone dogstatsd
		if awsEnv.DogstatsdDeploy() {
			if _, err := dogstatsdstandalone.K8sAppDefinition(*awsEnv.CommonEnvironment, eksKubeProvider, "dogstatsd-standalone", fakeIntake, true, ""); err != nil {
				return err
			}
		}

		// Deploy testing workload
		if awsEnv.TestingWorkloadDeploy() {
			if _, err := nginx.K8sAppDefinition(*awsEnv.CommonEnvironment, eksKubeProvider, "workload-nginx", dependsOnCrd); err != nil {
				return err
			}

			if _, err := redis.K8sAppDefinition(*awsEnv.CommonEnvironment, eksKubeProvider, "workload-redis", dependsOnCrd); err != nil {
				return err
			}

			if _, err := cpustress.K8sAppDefinition(*awsEnv.CommonEnvironment, eksKubeProvider, "workload-cpustress"); err != nil {
				return err
			}

			// dogstatsd clients that report to the Agent
			if _, err := dogstatsd.K8sAppDefinition(*awsEnv.CommonEnvironment, eksKubeProvider, "workload-dogstatsd", 8125, "/var/run/datadog/dsd.socket"); err != nil {
				return err
			}

			// dogstatsd clients that report to the dogstatsd standalone deployment
			if _, err := dogstatsd.K8sAppDefinition(*awsEnv.CommonEnvironment, eksKubeProvider, "workload-dogstatsd-standalone", dogstatsdstandalone.HostPort, dogstatsdstandalone.Socket); err != nil {
				return err
			}

			if _, err := prometheus.K8sAppDefinition(*awsEnv.CommonEnvironment, eksKubeProvider, "workload-prometheus"); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	return clusterComp.Export(ctx, nil)
}
