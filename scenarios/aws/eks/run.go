package eks

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components"
	"github.com/DataDog/test-infra-definitions/components/datadog/agent"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/cpustress"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/dogstatsd"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/mutatedbyadmissioncontroller"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/nginx"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/prometheus"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/redis"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/tracegen"
	dogstatsdstandalone "github.com/DataDog/test-infra-definitions/components/datadog/dogstatsd-standalone"
	fakeintakeComp "github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
	kubeComp "github.com/DataDog/test-infra-definitions/components/kubernetes"
	resourcesAws "github.com/DataDog/test-infra-definitions/resources/aws"
	localEks "github.com/DataDog/test-infra-definitions/resources/aws/eks"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/fakeintake"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	awsEks "github.com/pulumi/pulumi-aws/sdk/v6/go/aws/eks"
	awsIam "github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi-eks/sdk/v2/go/eks"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Run(ctx *pulumi.Context) error {
	awsEnv, err := resourcesAws.NewEnvironment(ctx)
	if err != nil {
		return err
	}

	clusterComp, err := components.NewComponent(&awsEnv, awsEnv.Namer.ResourceName("eks"), func(comp *kubeComp.Cluster) error {
		// Create Cluster SG
		clusterSG, err := ec2.NewSecurityGroup(ctx, awsEnv.Namer.ResourceName("eks-sg"), &ec2.SecurityGroupArgs{
			NamePrefix:  awsEnv.CommonNamer().DisplayName(255, pulumi.String("eks-sg")),
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
			Name:                         awsEnv.CommonNamer().DisplayName(100),
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
		}), awsEnv.WithProviders(config.ProviderEKS, config.ProviderAWS))
		if err != nil {
			return err
		}

		// Building Kubernetes provider
		eksKubeProvider, err := kubernetes.NewProvider(awsEnv.Ctx(), awsEnv.Namer.ResourceName("k8s-provider"), &kubernetes.ProviderArgs{
			Kubeconfig:            cluster.KubeconfigJson,
			EnableServerSideApply: pulumi.BoolPtr(true),
			DeleteUnreachable:     pulumi.BoolPtr(true),
		}, awsEnv.WithProviders(config.ProviderAWS))
		if err != nil {
			return err
		}

		// Filling Kubernetes component from EKS cluster
		comp.ClusterName = cluster.EksCluster.Name()
		comp.KubeConfig = cluster.KubeconfigJson
		comp.KubeProvider = eksKubeProvider

		// Create configuration for POD subnets if any
		workloadDeps := make([]pulumi.Resource, 0)
		if podSubnets := awsEnv.EKSPODSubnets(); len(podSubnets) > 0 {
			eniConfigs, err := localEks.NewENIConfigs(awsEnv, comp.KubeProvider, podSubnets, awsEnv.DefaultSecurityGroups())
			if err != nil {
				return err
			}

			// Setting AWS_VPC_K8S_CNI_CUSTOM_NETWORK_CFG is mandatory for EKS CNI to work with ENIConfig CRD
			dsPatch, err := appsv1.NewDaemonSetPatch(awsEnv.Ctx(), awsEnv.Namer.ResourceName("eks-custom-network"), &appsv1.DaemonSetPatchArgs{
				Metadata: metav1.ObjectMetaPatchArgs{
					Namespace: pulumi.String("kube-system"),
					Name:      pulumi.String("aws-node"),
					Annotations: pulumi.StringMap{
						"pulumi.com/patchForce": pulumi.String("true"),
					},
				},
				Spec: appsv1.DaemonSetSpecPatchArgs{
					Template: corev1.PodTemplateSpecPatchArgs{
						Spec: corev1.PodSpecPatchArgs{
							Containers: corev1.ContainerPatchArray{
								corev1.ContainerPatchArgs{
									Name: pulumi.StringPtr("aws-node"),
									Env: corev1.EnvVarPatchArray{
										corev1.EnvVarPatchArgs{
											Name:  pulumi.String("AWS_VPC_K8S_CNI_CUSTOM_NETWORK_CFG"),
											Value: pulumi.String("true"),
										},
										corev1.EnvVarPatchArgs{
											Name:  pulumi.String("ENI_CONFIG_LABEL_DEF"),
											Value: pulumi.String("topology.kubernetes.io/zone"),
										},
										corev1.EnvVarPatchArgs{
											Name:  pulumi.String("ENABLE_PREFIX_DELEGATION"),
											Value: pulumi.String("true"),
										},
										corev1.EnvVarPatchArgs{
											Name:  pulumi.String("WARM_IP_TARGET"),
											Value: pulumi.String("1"),
										},
										corev1.EnvVarPatchArgs{
											Name:  pulumi.String("MINIMUM_IP_TARGET"),
											Value: pulumi.String("1"),
										},
									},
								},
							},
						},
					},
				},
			}, pulumi.Provider(eksKubeProvider), utils.PulumiDependsOn(eniConfigs))
			if err != nil {
				return err
			}

			workloadDeps = append(workloadDeps, eniConfigs, dsPatch)
		}

		// Create managed node groups
		if awsEnv.EKSLinuxNodeGroup() {
			ng, err := localEks.NewLinuxNodeGroup(awsEnv, cluster, linuxNodeRole)
			if err != nil {
				return err
			}
			workloadDeps = append(workloadDeps, ng)
		}

		if awsEnv.EKSLinuxARMNodeGroup() {
			ng, err := localEks.NewLinuxARMNodeGroup(awsEnv, cluster, linuxNodeRole)
			if err != nil {
				return err
			}
			workloadDeps = append(workloadDeps, ng)
		}

		if awsEnv.EKSBottlerocketNodeGroup() {
			ng, err := localEks.NewBottlerocketNodeGroup(awsEnv, cluster, linuxNodeRole)
			if err != nil {
				return err
			}
			workloadDeps = append(workloadDeps, ng)
		}

		// Create unmanaged node groups
		if awsEnv.EKSWindowsNodeGroup() {
			_, err := localEks.NewWindowsUnmanagedNodeGroup(awsEnv, cluster, windowsNodeRole)
			if err != nil {
				return err
			}
		}

		// Applying necessary Windows configuration if Windows nodes
		// Custom networking is not available for Windows nodes, using normal subnets IPs
		if awsEnv.EKSWindowsNodeGroup() {
			_, err := corev1.NewConfigMapPatch(awsEnv.Ctx(), awsEnv.Namer.ResourceName("eks-cni-cm"), &corev1.ConfigMapPatchArgs{
				Metadata: metav1.ObjectMetaPatchArgs{
					Namespace: pulumi.String("kube-system"),
					Name:      pulumi.String("amazon-vpc-cni"),
					Annotations: pulumi.StringMap{
						"pulumi.com/patchForce": pulumi.String("true"),
					},
				},
				Data: pulumi.StringMap{
					"enable-windows-ipam": pulumi.String("true"),
				},
			}, pulumi.Provider(eksKubeProvider))
			if err != nil {
				return err
			}
		}

		var fakeIntake *fakeintakeComp.Fakeintake
		if awsEnv.AgentUseFakeintake() {
			fakeIntakeOptions := []fakeintake.Option{
				fakeintake.WithCPU(1024),
				fakeintake.WithMemory(6144),
			}
			if awsEnv.InfraShouldDeployFakeintakeWithLB() {
				fakeIntakeOptions = append(fakeIntakeOptions, fakeintake.WithLoadBalancer())
			}

			if fakeIntake, err = fakeintake.NewECSFargateInstance(awsEnv, "ecs", fakeIntakeOptions...); err != nil {
				return err
			}
			if err := fakeIntake.Export(awsEnv.Ctx(), nil); err != nil {
				return err
			}
		}

		// Deploy the agent
		dependsOnSetup := utils.PulumiDependsOn(workloadDeps...)
		var dependsOnCrd pulumi.ResourceOption
		if awsEnv.AgentDeploy() {
			helmComponent, err := agent.NewHelmInstallation(&awsEnv, agent.HelmInstallationArgs{
				KubeProvider:  eksKubeProvider,
				Namespace:     "datadog",
				Fakeintake:    fakeIntake,
				DeployWindows: awsEnv.EKSWindowsNodeGroup(),
			}, dependsOnSetup)
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
			if _, err := dogstatsdstandalone.K8sAppDefinition(&awsEnv, eksKubeProvider, "dogstatsd-standalone", fakeIntake, true, "", dependsOnSetup); err != nil {
				return err
			}
		}

		// Deploy testing workload
		if awsEnv.TestingWorkloadDeploy() {
			if _, err := nginx.K8sAppDefinition(&awsEnv, eksKubeProvider, "workload-nginx", "", dependsOnCrd); err != nil {
				return err
			}

			if _, err := redis.K8sAppDefinition(&awsEnv, eksKubeProvider, "workload-redis", dependsOnCrd); err != nil {
				return err
			}

			if _, err := cpustress.K8sAppDefinition(&awsEnv, eksKubeProvider, "workload-cpustress", dependsOnSetup); err != nil {
				return err
			}

			// dogstatsd clients that report to the Agent
			if _, err := dogstatsd.K8sAppDefinition(&awsEnv, eksKubeProvider, "workload-dogstatsd", 8125, "/var/run/datadog/dsd.socket", dependsOnSetup); err != nil {
				return err
			}

			// dogstatsd clients that report to the dogstatsd standalone deployment
			if _, err := dogstatsd.K8sAppDefinition(&awsEnv, eksKubeProvider, "workload-dogstatsd-standalone", dogstatsdstandalone.HostPort, dogstatsdstandalone.Socket, dependsOnSetup); err != nil {
				return err
			}

			if _, err := tracegen.K8sAppDefinition(&awsEnv, eksKubeProvider, "workload-tracegen", dependsOnSetup); err != nil {
				return err
			}

			if _, err := prometheus.K8sAppDefinition(&awsEnv, eksKubeProvider, "workload-prometheus", dependsOnSetup); err != nil {
				return err
			}

			if _, err := mutatedbyadmissioncontroller.K8sAppDefinition(&awsEnv, eksKubeProvider, "workload-mutated", "workload-mutated-lib-injection", dependsOnSetup); err != nil {
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
