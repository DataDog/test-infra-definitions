package eks

import (
	"github.com/DataDog/test-infra-definitions/aws"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	awsEks "github.com/pulumi/pulumi-aws/sdk/v5/go/aws/eks"
	awsIam "github.com/pulumi/pulumi-aws/sdk/v5/go/aws/iam"
	"github.com/pulumi/pulumi-eks/sdk/go/eks"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Run(ctx *pulumi.Context) error {
	awsEnv, err := aws.AWSEnvironment(ctx)
	if err != nil {
		return err
	}

	// Create Cluster SG
	clusterSG, err := ec2.NewSecurityGroup(ctx, ctx.Stack()+"-eks-sg", &ec2.SecurityGroupArgs{
		NamePrefix:  pulumi.StringPtr(ctx.Stack() + "-eks-sg"),
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
	})
	if err != nil {
		return err
	}

	// IAM Node role
	linuxNodeRole, err := GetNodeRole(awsEnv, "linux-node-role")
	if err != nil {
		return err
	}

	windowsNodeRole, err := GetNodeRole(awsEnv, "windows-node-role")
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
	cluster, err := eks.NewCluster(ctx, ctx.Stack(), &eks.ClusterArgs{
		Name:                         pulumi.StringPtr(ctx.Stack()),
		Version:                      pulumi.StringPtr(awsEnv.KubernetesVersion()),
		EndpointPrivateAccess:        pulumi.BoolPtr(true),
		EndpointPublicAccess:         pulumi.BoolPtr(false),
		Fargate:                      fargateProfile,
		ClusterSecurityGroup:         clusterSG,
		NodeAssociatePublicIpAddress: pulumi.BoolPtr(false),
		PrivateSubnetIds:             pulumi.ToStringArray(awsEnv.DefaultSubnets()),
		VpcId:                        pulumi.StringPtr(awsEnv.DefaultVPCID()),
		SkipDefaultNodeGroup:         pulumi.BoolPtr(true),
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
	})
	if err != nil {
		return err
	}

	// Create managed node groups
	if awsEnv.EKSLinuxNodeGroup() {
		_, err := NewLinuxNodeGroup(awsEnv, cluster, linuxNodeRole)
		if err != nil {
			return err
		}
	}

	if awsEnv.EKSLinuxARMNodeGroup() {
		_, err := NewLinuxARMNodeGroup(awsEnv, cluster, linuxNodeRole)
		if err != nil {
			return err
		}
	}

	if awsEnv.EKSBottlerocketNodeGroup() {
		_, err := NewBottlerocketNodeGroup(awsEnv, cluster, linuxNodeRole)
		if err != nil {
			return err
		}
	}

	// Create unmanaged node groups
	if awsEnv.EKSWindowsNodeGroup() {
		_, err := NewWindowsUnmanagedNodeGroup(awsEnv, cluster, windowsNodeRole)
		if err != nil {
			return err
		}
	}

	// Export the cluster's kubeconfig.
	ctx.Export("kubeconfig", cluster.Kubeconfig)
	return nil
}
