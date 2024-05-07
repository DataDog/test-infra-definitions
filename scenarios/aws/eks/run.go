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
	v1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/rbac/v1"
	"github.com/pulumi/pulumi-random/sdk/v4/go/random"
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
			InstanceRoles: awsIam.RoleArray{
				linuxNodeRole,
				windowsNodeRole,
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

		// Deps for nodes and workloads
		nodeDeps := make([]pulumi.Resource, 0)
		workloadDeps := make([]pulumi.Resource, 0)

		// Create configuration for POD subnets if any
		if podSubnets := awsEnv.EKSPODSubnets(); len(podSubnets) > 0 {
			eniConfigs, err := localEks.NewENIConfigs(awsEnv, podSubnets, awsEnv.DefaultSecurityGroups(), pulumi.Provider(eksKubeProvider))
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

			nodeDeps = append(nodeDeps, eniConfigs, dsPatch)
		}

		// Create managed node groups
		if awsEnv.EKSLinuxNodeGroup() {
			ng, err := localEks.NewLinuxNodeGroup(awsEnv, cluster, linuxNodeRole, utils.PulumiDependsOn(nodeDeps...))
			if err != nil {
				return err
			}
			workloadDeps = append(workloadDeps, ng)
		}

		if awsEnv.EKSLinuxARMNodeGroup() {
			ng, err := localEks.NewLinuxARMNodeGroup(awsEnv, cluster, linuxNodeRole, utils.PulumiDependsOn(nodeDeps...))
			if err != nil {
				return err
			}
			workloadDeps = append(workloadDeps, ng)
		}

		if awsEnv.EKSBottlerocketNodeGroup() {
			ng, err := localEks.NewBottlerocketNodeGroup(awsEnv, cluster, linuxNodeRole, utils.PulumiDependsOn(nodeDeps...))
			if err != nil {
				return err
			}
			workloadDeps = append(workloadDeps, ng)
		}

		if awsEnv.EKSWindowsNodeGroup() {
			// Applying necessary Windows configuration if Windows nodes
			// Custom networking is not available for Windows nodes, using normal subnets IPs
			winCNIPatch, err := corev1.NewConfigMapPatch(awsEnv.Ctx(), awsEnv.Namer.ResourceName("eks-cni-cm"), &corev1.ConfigMapPatchArgs{
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

			nodeDeps = append(nodeDeps, winCNIPatch)
			ng, err := localEks.NewWindowsNodeGroup(awsEnv, cluster, windowsNodeRole, utils.PulumiDependsOn(nodeDeps...))
			if err != nil {
				return err
			}
			workloadDeps = append(workloadDeps, ng)
		}

		// Create fakeintake if needed
		var fakeIntake *fakeintakeComp.Fakeintake
		if awsEnv.AgentUseFakeintake() {
			fakeIntakeOptions := []fakeintake.Option{
				fakeintake.WithMemory(2048),
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

		randomClusterAgentToken, err := random.NewRandomString(awsEnv.CommonEnvironment.Ctx(), "datadog-cluster-agent-token", &random.RandomStringArgs{
			Lower:   pulumi.Bool(true),
			Upper:   pulumi.Bool(true),
			Length:  pulumi.Int(32),
			Numeric: pulumi.Bool(false),
			Special: pulumi.Bool(false),
		}, pulumi.Providers(eksKubeProvider), awsEnv.CommonEnvironment.WithProviders(config.ProviderRandom), pulumi.Parent(eksKubeProvider), pulumi.DeletedWith(eksKubeProvider))
		if err != nil {
			return err
		}

		// Deploy the agent
		workloadWithCRDDeps := make([]pulumi.Resource, 0, len(workloadDeps))
		copy(workloadWithCRDDeps, workloadDeps)

		if awsEnv.AgentDeploy() {
			fargateInjectionCustomValues := `
clusterAgent:
  admissionController:
    agentSidecarInjection:
      enabled: true
      provider: fargate
`

			helmComponent, err := agent.NewHelmInstallation(&awsEnv, agent.HelmInstallationArgs{
				KubeProvider:                   eksKubeProvider,
				Namespace:                      "datadog",
				ValuesYAML:                     pulumi.AssetOrArchiveArray{pulumi.NewStringAsset(fargateInjectionCustomValues)},
				Fakeintake:                     fakeIntake,
				DeployWindows:                  awsEnv.EKSWindowsNodeGroup(),
				ClusterAgentToken:              randomClusterAgentToken,
				EnableSidecarProfileFakeIntake: true,
			}, utils.PulumiDependsOn(workloadDeps...))
			if err != nil {
				return err
			}

			ctx.Export("agent-linux-helm-install-name", helmComponent.LinuxHelmReleaseName)
			ctx.Export("agent-linux-helm-install-status", helmComponent.LinuxHelmReleaseStatus)
			if awsEnv.EKSWindowsNodeGroup() {
				ctx.Export("agent-windows-helm-install-name", helmComponent.WindowsHelmReleaseName)
				ctx.Export("agent-windows-helm-install-status", helmComponent.WindowsHelmReleaseStatus)
			}

			workloadWithCRDDeps = append(workloadWithCRDDeps, helmComponent)
		}

		// Deploy standalone dogstatsd
		if awsEnv.DogstatsdDeploy() {
			if _, err := dogstatsdstandalone.K8sAppDefinition(&awsEnv, eksKubeProvider, "dogstatsd-standalone", fakeIntake, true, "", utils.PulumiDependsOn(workloadDeps...)); err != nil {
				return err
			}
		}

		// Deploy testing workload
		if awsEnv.TestingWorkloadDeploy() {
			if _, err := nginx.K8sAppDefinition(&awsEnv, eksKubeProvider, "workload-nginx", "", true, utils.PulumiDependsOn(workloadWithCRDDeps...)); err != nil {
				return err
			}

			if _, err := redis.K8sAppDefinition(&awsEnv, eksKubeProvider, "workload-redis", true, utils.PulumiDependsOn(workloadWithCRDDeps...)); err != nil {
				return err
			}

			if _, err := cpustress.K8sAppDefinition(&awsEnv, eksKubeProvider, "workload-cpustress", utils.PulumiDependsOn(workloadDeps...)); err != nil {
				return err
			}

			// dogstatsd clients that report to the Agent
			if _, err := dogstatsd.K8sAppDefinition(&awsEnv, eksKubeProvider, "workload-dogstatsd", 8125, "/var/run/datadog/dsd.socket", utils.PulumiDependsOn(workloadDeps...)); err != nil {
				return err
			}

			// dogstatsd clients that report to the dogstatsd standalone deployment
			if _, err := dogstatsd.K8sAppDefinition(&awsEnv, eksKubeProvider, "workload-dogstatsd-standalone", dogstatsdstandalone.HostPort, dogstatsdstandalone.Socket, utils.PulumiDependsOn(workloadDeps...)); err != nil {
				return err
			}

			if _, err := tracegen.K8sAppDefinition(&awsEnv, eksKubeProvider, "workload-tracegen", utils.PulumiDependsOn(workloadDeps...)); err != nil {
				return err
			}

			if _, err := prometheus.K8sAppDefinition(&awsEnv, eksKubeProvider, "workload-prometheus", utils.PulumiDependsOn(workloadDeps...)); err != nil {
				return err
			}

			if _, err := mutatedbyadmissioncontroller.K8sAppDefinition(&awsEnv, eksKubeProvider, "workload-mutated", "workload-mutated-lib-injection", utils.PulumiDependsOn(workloadDeps...)); err != nil {
				return err
			}

			if fargateNamespace := awsEnv.EKSFargateNamespace(); fargateNamespace != "" {
				fgns, err := corev1.NewNamespace(awsEnv.CommonEnvironment.Ctx(), "fargate", &corev1.NamespaceArgs{
					Metadata: metav1.ObjectMetaArgs{
						Name: pulumi.String("fargate"),
					},
				}, pulumi.Provider(eksKubeProvider), pulumi.Parent(eksKubeProvider), pulumi.DeletedWith(eksKubeProvider))
				if err != nil {
					return err
				}
				dependsOnFargate := utils.PulumiDependsOn(fgns)

				fargateSecret, err := corev1.NewSecret(awsEnv.CommonEnvironment.Ctx(), "datadog-credentials-injection", &corev1.SecretArgs{
					Metadata: metav1.ObjectMetaArgs{
						Namespace: pulumi.StringPtr("fargate"),
						Name:      pulumi.Sprintf("datadog-secret"),
					},
					StringData: pulumi.StringMap{
						"api-key": awsEnv.CommonEnvironment.AgentAPIKey(),
						"app-key": awsEnv.CommonEnvironment.AgentAPPKey(),
						"token":   randomClusterAgentToken.Result,
					},
				}, pulumi.Providers(eksKubeProvider), pulumi.Parent(eksKubeProvider), pulumi.DeletedWith(eksKubeProvider), dependsOnFargate)
				if err != nil {
					return err
				}
				dependsOnSecret := utils.PulumiDependsOn(fargateSecret)

				clusterRoleArgs := v1.ClusterRoleArgs{
					Metadata: &metav1.ObjectMetaArgs{
						Name:      pulumi.String("datadog-agent"),
						Namespace: pulumi.String("fargate"),
					},
					Rules: v1.PolicyRuleArray{
						v1.PolicyRuleArgs{
							ApiGroups: pulumi.StringArray{
								pulumi.String(""),
							},
							Resources: pulumi.StringArray{
								pulumi.String("nodes"),
								pulumi.String("namespaces"),
								pulumi.String("endpoints"),
							},
							Verbs: pulumi.StringArray{
								pulumi.String("get"),
								pulumi.String("list"),
							},
						},
						v1.PolicyRuleArgs{
							ApiGroups: pulumi.StringArray{
								pulumi.String(""),
							},
							Resources: pulumi.StringArray{
								pulumi.String("nodes/metrics"),
								pulumi.String("nodes/spec"),
								pulumi.String("nodes/stats"),
								pulumi.String("nodes/proxy"),
								pulumi.String("nodes/pods"),
								pulumi.String("nodes/healthz"),
							},
							Verbs: pulumi.StringArray{
								pulumi.String("get"),
							},
						},
					},
				}

				clusterRoleBindingArgs := v1.ClusterRoleBindingArgs{
					Metadata: &metav1.ObjectMetaArgs{
						Name: pulumi.String("datadog-agent"),
					},
					RoleRef: v1.RoleRefArgs{
						ApiGroup: pulumi.String("rbac.authorization.k8s.io"),
						Kind:     pulumi.String("ClusterRole"),
						Name:     pulumi.String("datadog-agent"),
					},
					Subjects: v1.SubjectArray{
						&v1.SubjectArgs{
							Kind:      pulumi.String("ServiceAccount"),
							Name:      pulumi.String("datadog-agent"),
							Namespace: pulumi.String(fargateNamespace),
						},
					},
				}

				serviceAccountArgs := corev1.ServiceAccountArgs{
					Metadata: &metav1.ObjectMetaArgs{
						Name:      pulumi.String("datadog-agent"),
						Namespace: pulumi.String(fargateNamespace),
					},
				}

				if _, err := v1.NewClusterRole(awsEnv.CommonEnvironment.Ctx(), "datadog-agent", &clusterRoleArgs, pulumi.Providers(eksKubeProvider), pulumi.Parent(eksKubeProvider), pulumi.DeletedWith(eksKubeProvider)); err != nil {
					return err
				}

				if _, err := v1.NewClusterRoleBinding(awsEnv.CommonEnvironment.Ctx(), "datadog-agent", &clusterRoleBindingArgs, pulumi.Providers(eksKubeProvider), pulumi.Parent(eksKubeProvider), pulumi.DeletedWith(eksKubeProvider)); err != nil {
					return err
				}

				if _, err := corev1.NewServiceAccount(awsEnv.CommonEnvironment.Ctx(), "datadog-agent", &serviceAccountArgs, pulumi.Providers(eksKubeProvider), pulumi.Parent(eksKubeProvider), pulumi.DeletedWith(eksKubeProvider)); err != nil {
					return err
				}

				if _, err := nginx.EKSFargateAppDefinition(&awsEnv, fargateNamespace, true, pulumi.Providers(eksKubeProvider), pulumi.Parent(eksKubeProvider), pulumi.DeletedWith(eksKubeProvider), dependsOnFargate, utils.PulumiDependsOn(workloadWithCRDDeps...), dependsOnSecret); err != nil {
					return err
				}

				if _, err := redis.EKSFargateAppDefinition(&awsEnv, fargateNamespace, true, pulumi.Providers(eksKubeProvider), pulumi.Parent(eksKubeProvider), pulumi.DeletedWith(eksKubeProvider), dependsOnFargate, utils.PulumiDependsOn(workloadWithCRDDeps...), dependsOnSecret); err != nil {
					return err
				}
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	return clusterComp.Export(ctx, nil)
}
