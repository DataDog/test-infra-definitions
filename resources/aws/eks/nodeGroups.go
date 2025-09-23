package eks

import (
	"strings"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/resources/aws"

	awsEc2 "github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	awsEks "github.com/pulumi/pulumi-aws/sdk/v6/go/aws/eks"
	awsIam "github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi-eks/sdk/v3/go/eks"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	amazonLinux2AMD64AmiType    = "AL2_x86_64"
	amazonLinux2ARM64AmiType    = "AL2_ARM_64"
	amazonLinux2023AMD64AmiType = "AL2023_x86_64_STANDARD"
	amazonLinux2023ARM64AmiType = "AL2023_ARM_64_STANDARD"
	bottlerocketAmiType         = "BOTTLEROCKET_x86_64"
	windowsAmiType              = "WINDOWS_CORE_2022_x86_64"
)

func NewLinuxNodeGroup(e aws.Environment, cluster *eks.Cluster, nodeRole *awsIam.Role, opts ...pulumi.ResourceOption) (*eks.ManagedNodeGroup, error) {
	return newManagedNodeGroup(e, "linux", cluster, nodeRole, amazonLinux2AMD64AmiType, e.DefaultInstanceType(), opts...)
}

func NewAL2023LinuxNodeGroup(e aws.Environment, cluster *eks.Cluster, nodeRole *awsIam.Role, opts ...pulumi.ResourceOption) (*eks.ManagedNodeGroup, error) {
	return newManagedAL2023NodeGroup(e, "linux", cluster, nodeRole, amazonLinux2023AMD64AmiType, e.DefaultInstanceType(), opts...)
}

func NewAL2023LinuxARMNodeGroup(e aws.Environment, cluster *eks.Cluster, nodeRole *awsIam.Role, opts ...pulumi.ResourceOption) (*eks.ManagedNodeGroup, error) {
	return newManagedNodeGroup(e, "linux-arm", cluster, nodeRole, amazonLinux2023ARM64AmiType, e.DefaultARMInstanceType(), opts...)
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

func newManagedAL2023NodeGroup(e aws.Environment, name string, cluster *eks.Cluster, nodeRole *awsIam.Role, amiType, instanceType string, opts ...pulumi.ResourceOption) (*eks.ManagedNodeGroup, error) {
	launchTemplateName := name + "-launch-template"
	securityGroupName := launchTemplateName + "-security-group"

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

	nodeGroupInstanceSG, err := awsEc2.NewSecurityGroup(e.Ctx(), securityGroupName, &awsEc2.SecurityGroupArgs{
		Description: pulumi.String("Allow SSH to EKS nodes from source security groups"),
		VpcId:       pulumi.StringPtr(e.DefaultVPCID()),
		// No inline ingress here; we’ll add one rule per source SG below.
		Egress: awsEc2.SecurityGroupEgressArray{
			&awsEc2.SecurityGroupEgressArgs{ // allow all egress (typical)
				Protocol:   pulumi.String("-1"),
				FromPort:   pulumi.Int(0),
				ToPort:     pulumi.Int(0),
				CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
			&awsEc2.SecurityGroupEgressArgs{
				Protocol:       pulumi.String("-1"), // all protocols
				FromPort:       pulumi.Int(0),
				ToPort:         pulumi.Int(0),
				Ipv6CidrBlocks: pulumi.StringArray{pulumi.String("::/0")},
			},
		},
	}, utils.MergeOptions(opts, e.WithProviders(config.ProviderAWS, config.ProviderEKS))...)
	if err != nil {
		return nil, err
	}

	for _, src := range e.EKSAllowedInboundSecurityGroups() {
		_, err := awsEc2.NewSecurityGroupRule(e.Ctx(), "ingress-from-"+src, &awsEc2.SecurityGroupRuleArgs{
			Type:                  pulumi.String("ingress"),
			SecurityGroupId:       nodeGroupInstanceSG.ID(),
			FromPort:              pulumi.Int(22),
			ToPort:                pulumi.Int(22),
			Protocol:              pulumi.String("tcp"),
			SourceSecurityGroupId: pulumi.String(src),
		}, utils.MergeOptions(opts, e.WithProviders(config.ProviderAWS, config.ProviderEKS))...)
		if err != nil {
			return nil, err
		}
	}

	lt, err := awsEc2.NewLaunchTemplate(e.Ctx(), launchTemplateName, &awsEc2.LaunchTemplateArgs{
		UpdateDefaultVersion: pulumi.BoolPtr(true),
		KeyName:              pulumi.String(e.DefaultKeyPairName()),
		MetadataOptions: &awsEc2.LaunchTemplateMetadataOptionsArgs{
			HttpPutResponseHopLimit: pulumi.IntPtr(2),
		},
		BlockDeviceMappings: awsEc2.LaunchTemplateBlockDeviceMappingArray{
			&awsEc2.LaunchTemplateBlockDeviceMappingArgs{
				DeviceName: pulumi.String("/dev/xvda"),
				Ebs: &awsEc2.LaunchTemplateBlockDeviceMappingEbsArgs{
					VolumeSize:          pulumi.Int(80),
					VolumeType:          pulumi.String("gp3"),
					DeleteOnTermination: pulumi.String("true"),
					Encrypted:           pulumi.String("false"),
				},
			},
		},
		VpcSecurityGroupIds: append(pulumi.StringArray{nodeGroupInstanceSG.ID()}, pulumi.ToStringArray(e.DefaultSecurityGroups())...),
	}, utils.MergeOptions(opts, e.WithProviders(config.ProviderAWS, config.ProviderEKS))...)

	if err != nil {
		return nil, err
	}

	return eks.NewManagedNodeGroup(e.Ctx(), e.Namer.ResourceName(name), &eks.ManagedNodeGroupArgs{
		AmiType:             pulumi.StringPtr(amiType),
		Cluster:             cluster.Core,
		InstanceTypes:       pulumi.ToStringArray([]string{instanceType}),
		ForceUpdateVersion:  pulumi.BoolPtr(true),
		NodeGroupNamePrefix: e.CommonNamer().DisplayName(37, pulumi.String(name), pulumi.String("ng")),
		ScalingConfig: awsEks.NodeGroupScalingConfigArgs{
			DesiredSize: pulumi.Int(1),
			MaxSize:     pulumi.Int(1),
			MinSize:     pulumi.Int(0),
		},
		NodeRole: nodeRole,
		Taints:   taints,
		LaunchTemplate: &awsEks.NodeGroupLaunchTemplateArgs{
			Id:      lt.ID(),
			Version: pulumi.String("$Latest"),
		},
	}, utils.MergeOptions(opts, e.WithProviders(config.ProviderAWS, config.ProviderEKS))...)
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
