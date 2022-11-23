package eks

import (
	"encoding/base64"
	"fmt"

	"github.com/DataDog/test-infra-definitions/aws"

	awsEks "github.com/pulumi/pulumi-aws/sdk/v5/go/aws/eks"
	awsIam "github.com/pulumi/pulumi-aws/sdk/v5/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ssm"
	"github.com/pulumi/pulumi-eks/sdk/go/eks"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	amazonLinux2AMD64AmiType = "AL2_x86_64"
	amazonLinux2ARM64AmiType = "AL2_ARM_64"
	bottlerocketAmiType      = "BOTTLEROCKET_x86_64"

	windowsInitUserData = `<powershell>
[string]$EKSBootstrapScriptFile = "$env:ProgramFiles\Amazon\EKS\Start-EKSBootstrap.ps1"
& $EKSBootstrapScriptFile -EKSClusterName %s 3>&1 4>&1 5>&1 6>&1
</powershell>
`
)

func NewLinuxNodeGroup(e aws.Environment, cluster *eks.Cluster, nodeRole *awsIam.Role) (*eks.ManagedNodeGroup, error) {
	return newManagedNodeGroup(e, "linux-ng", cluster, nodeRole, amazonLinux2AMD64AmiType, e.DefaultInstanceType())
}

func NewLinuxARMNodeGroup(e aws.Environment, cluster *eks.Cluster, nodeRole *awsIam.Role) (*eks.ManagedNodeGroup, error) {
	return newManagedNodeGroup(e, "linux-arm-ng", cluster, nodeRole, amazonLinux2ARM64AmiType, e.DefaultARMInstanceType())
}

func NewBottlerocketNodeGroup(e aws.Environment, cluster *eks.Cluster, nodeRole *awsIam.Role) (*eks.ManagedNodeGroup, error) {
	return newManagedNodeGroup(e, "bottlerocket-ng", cluster, nodeRole, bottlerocketAmiType, e.DefaultInstanceType())
}

func newManagedNodeGroup(e aws.Environment, name string, cluster *eks.Cluster, nodeRole *awsIam.Role, amiType, instanceType string) (*eks.ManagedNodeGroup, error) {
	return eks.NewManagedNodeGroup(e.Ctx, e.Namer.ResourceName(name), &eks.ManagedNodeGroupArgs{
		AmiType:             pulumi.StringPtr(amiType),
		Cluster:             cluster.Core,
		DiskSize:            pulumi.Int(80),
		InstanceTypes:       pulumi.ToStringArray([]string{instanceType}),
		ForceUpdateVersion:  pulumi.BoolPtr(true),
		NodeGroupNamePrefix: pulumi.String(e.Ctx.Stack() + "-" + name),
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
	}, pulumi.Provider(e.Provider))
}

func NewWindowsUnmanagedNodeGroup(e aws.Environment, cluster *eks.Cluster, nodeRole *awsIam.Role) (*eks.NodeGroup, error) {
	// Currently only Windows 2019 is supported on EKS (as opposed to ECS)
	windowsAmi, err := ssm.LookupParameter(e.Ctx, &ssm.LookupParameterArgs{
		Name: fmt.Sprintf("/aws/service/ami-windows-latest/Windows_Server-2019-English-Core-EKS_Optimized-%s/image_id", e.KubernetesVersion()),
	}, pulumi.Provider(e.Provider))
	if err != nil {
		return nil, err
	}

	return newUnmanagedNodeGroup(e, "windows-ng", cluster, nodeRole, pulumi.String(windowsAmi.Value), pulumi.String(e.DefaultInstanceType()), getUserData(windowsInitUserData, cluster.EksCluster.Name()))
}

func newUnmanagedNodeGroup(e aws.Environment, name string, cluster *eks.Cluster, nodeRole *awsIam.Role, ami, instanceType, userData pulumi.StringInput) (*eks.NodeGroup, error) {
	instanceProfile, err := awsIam.NewInstanceProfile(e.Ctx, e.Namer.ResourceName(name), &awsIam.InstanceProfileArgs{
		Name: e.Namer.DisplayName(pulumi.String(name)),
		Role: nodeRole.Name,
	}, pulumi.Provider(e.Provider))
	if err != nil {
		return nil, err
	}

	return eks.NewNodeGroup(e.Ctx, e.Namer.ResourceName(name), &eks.NodeGroupArgs{
		NodeUserDataOverride: userData,
		Cluster:              cluster.Core,
		DesiredCapacity:      pulumi.Int(1),
		// Currently not working
		// ExtraNodeSecurityGroups: extraSecurityGroups,
		KeyName:                      pulumi.StringPtr(e.DefaultKeyPairName()),
		AmiId:                        ami,
		InstanceType:                 instanceType,
		NodeRootVolumeSize:           pulumi.Int(80),
		NodeAssociatePublicIpAddress: pulumi.BoolPtr(false),
		InstanceProfile:              instanceProfile,
	}, pulumi.Provider(e.Provider))
}

func getUserData(userData string, clusterName pulumi.StringInput) pulumi.StringInput {
	return clusterName.ToStringOutput().ApplyT(func(name string) pulumi.StringInput {
		return pulumi.String(base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf(userData, name))))
	}).(pulumi.StringInput)
}
