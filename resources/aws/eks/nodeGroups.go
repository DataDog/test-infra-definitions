package eks

import (
	"strings"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/resources/aws"

	awsEks "github.com/pulumi/pulumi-aws/sdk/v6/go/aws/eks"
	awsIam "github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi-eks/sdk/v3/go/eks"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	amazonLinux2AMD64AmiType = "AL2023_x86_64_STANDARD"
	amazonLinux2ARM64AmiType = "AL2023_ARM_64_STANDARD"
	bottlerocketAmiType      = "BOTTLEROCKET_x86_64"
	windowsAmiType           = "WINDOWS_CORE_2022_x86_64"
)

func NewLinuxNodeGroup(e aws.Environment, cluster *eks.Cluster, nodeRole *awsIam.Role, opts ...pulumi.ResourceOption) (*eks.ManagedNodeGroup, error) {
	return newManagedNodeGroup(e, "linux", cluster, nodeRole, amazonLinux2AMD64AmiType, e.DefaultInstanceType(), opts...)
}

func NewLinuxARMNodeGroup(e aws.Environment, cluster *eks.Cluster, nodeRole *awsIam.Role, opts ...pulumi.ResourceOption) (*eks.ManagedNodeGroup, error) {
	return newManagedNodeGroup(e, "linux-arm", cluster, nodeRole, amazonLinux2ARM64AmiType, e.DefaultARMInstanceType(), opts...)
}

func NewBottlerocketNodeGroup(e aws.Environment, cluster *eks.Cluster, nodeRole *awsIam.Role, opts ...pulumi.ResourceOption) (*eks.ManagedNodeGroup, error) {
	return newManagedNodeGroup(e, "bottlerocket", cluster, nodeRole, bottlerocketAmiType, e.DefaultInstanceType(), opts...)
}

func NewWindowsNodeGroup(e aws.Environment, cluster *eks.Cluster, nodeRole *awsIam.Role, opts ...pulumi.ResourceOption) (*eks.ManagedNodeGroup, error) {
	return newManagedNodeGroup(e, "windows", cluster, nodeRole, windowsAmiType, e.DefaultInstanceType(), opts...)
}

func newManagedNodeGroup(e aws.Environment, name string, cluster *eks.Cluster, nodeRole *awsIam.Role, amiType, instanceType string, opts ...pulumi.ResourceOption) (*eks.ManagedNodeGroup, error) {
	taints := awsEks.NodeGroupTaintArray{}
	if strings.Contains(amiType, "WINDOWS") {
		taints = append(taints,
			awsEks.NodeGroupTaintArgs{
				Key:    pulumi.String("node.kubernetes.io/os"),
				Value:  pulumi.String("windows"),
				Effect: pulumi.String("NO_SCHEDULE"),
			},
		)
	}

	return eks.NewManagedNodeGroup(e.Ctx(), e.Namer.ResourceName(name), &eks.ManagedNodeGroupArgs{
		AmiType:             pulumi.StringPtr(amiType),
		Cluster:             cluster.Core,
		DiskSize:            pulumi.Int(80),
		InstanceTypes:       pulumi.ToStringArray([]string{instanceType}),
		ForceUpdateVersion:  pulumi.BoolPtr(true),
		NodeGroupNamePrefix: e.CommonNamer().DisplayName(37, pulumi.String(name), pulumi.String("ng")),
		ScalingConfig: awsEks.NodeGroupScalingConfigArgs{
			DesiredSize: pulumi.Int(1),
			MaxSize:     pulumi.Int(1),
			MinSize:     pulumi.Int(0),
		},
		NodeRole: nodeRole,
		RemoteAccess: awsEks.NodeGroupRemoteAccessArgs{
			Ec2SshKey:              pulumi.String(e.DefaultKeyPairName()),
			SourceSecurityGroupIds: pulumi.ToStringArray(e.EKSAllowedInboundSecurityGroups()),
		},
		Taints: taints,
	}, utils.MergeOptions(opts, e.WithProviders(config.ProviderAWS, config.ProviderEKS))...)
}
