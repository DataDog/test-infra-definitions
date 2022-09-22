package eks

import (
	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/aws/iam"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	awsEks "github.com/pulumi/pulumi-aws/sdk/v5/go/aws/eks"
	awsIam "github.com/pulumi/pulumi-aws/sdk/v5/go/aws/iam"
	"github.com/pulumi/pulumi-eks/sdk/go/eks"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	eksFargateNamespace = "fargate-workload"
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
	assumeRolePolicy, err := iam.GetAWSPrincipalAssumeRole(awsEnv)
	if err != nil {
		return err
	}

	nodeRole, err := awsIam.NewRole(ctx, ctx.Stack()+"-node-role", &awsIam.RoleArgs{
		NamePrefix:          pulumi.StringPtr(ctx.Stack() + "-node-role"),
		Description:         pulumi.StringPtr("Node role for EKS Cluster: " + ctx.Stack()),
		ForceDetachPolicies: pulumi.BoolPtr(true),
		ManagedPolicyArns: pulumi.ToStringArray([]string{
			"arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy",
			"arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly",
			"arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy",
		}),
		AssumeRolePolicy: pulumi.String(assumeRolePolicy.Json),
	})
	if err != nil {
		return err
	}

	// Create an EKS cluster with the default configuration.
	cluster, err := eks.NewCluster(ctx, ctx.Stack(), &eks.ClusterArgs{
		Name:                  pulumi.StringPtr(ctx.Stack()),
		EndpointPrivateAccess: pulumi.BoolPtr(true),
		EndpointPublicAccess:  pulumi.BoolPtr(false),
		Fargate: pulumi.Any(
			eks.FargateProfile{
				Selectors: []awsEks.FargateProfileSelector{
					{
						Namespace: eksFargateNamespace,
					},
				},
			},
		),
		ClusterSecurityGroup:         clusterSG,
		NodeAssociatePublicIpAddress: pulumi.BoolPtr(false),
		PrivateSubnetIds:             pulumi.ToStringArray(awsEnv.DefaultSubnets()),
		VpcId:                        pulumi.StringPtr(awsEnv.DefaultVPCID()),
		SkipDefaultNodeGroup:         pulumi.BoolPtr(true),
		InstanceRole:                 nodeRole,
	})
	if err != nil {
		return err
	}

	// Creating managed node group: only for AL2 Nodes
	_, err = eks.NewManagedNodeGroup(ctx, ctx.Stack()+"-linux-ng", &eks.ManagedNodeGroupArgs{
		AmiType:             pulumi.String("AL2_x86_64"),
		Cluster:             cluster.Core,
		DiskSize:            pulumi.Int(50),
		InstanceTypes:       pulumi.ToStringArray([]string{awsEnv.DefaultInstanceType()}),
		ForceUpdateVersion:  pulumi.BoolPtr(true),
		NodeGroupNamePrefix: pulumi.String(ctx.Stack() + "-linux-ng"),
		ScalingConfig: awsEks.NodeGroupScalingConfigArgs{
			DesiredSize: pulumi.Int(2),
			MaxSize:     pulumi.Int(2),
			MinSize:     pulumi.Int(0),
		},
		NodeRole: nodeRole,
		RemoteAccess: awsEks.NodeGroupRemoteAccessArgs{
			Ec2SshKey:              pulumi.String(awsEnv.DefaultKeyPairName()),
			SourceSecurityGroupIds: pulumi.ToStringArray(awsEnv.EKSAllowedInboundSecurityGroups()),
		},
	})
	if err != nil {
		return err
	}

	// Export the cluster's kubeconfig.
	ctx.Export("kubeconfig", cluster.Kubeconfig)
	return nil
}
