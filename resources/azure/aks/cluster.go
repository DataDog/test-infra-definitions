package aks

import (
	"encoding/base64"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/resources/azure"

	"github.com/pulumi/pulumi-azure-native-sdk/containerservice"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	adminUsername = "azureuser"
)

func NewCluster(e azure.Environment, name string, nodePool containerservice.ManagedClusterAgentPoolProfileArray, opts ...pulumi.ResourceOption) (*containerservice.ManagedCluster, pulumi.StringOutput, error) {
	sshPublicKey, err := utils.GetSSHPublicKey(e.DefaultPublicKeyPath())
	if err != nil {
		return nil, pulumi.StringOutput{}, err
	}

	// Warning: we're modifying passed array as it should normally never be used anywhere else
	nodePool = append(nodePool, systemNodePool(e, "system"))

	opts = append(opts, e.WithProviders(config.ProviderAzure))
	cluster, err := containerservice.NewManagedCluster(e.Ctx, e.Namer.ResourceName(name), &containerservice.ManagedClusterArgs{
		ResourceName:      e.CommonNamer.DisplayName(pulumi.String(name)),
		ResourceGroupName: pulumi.String(e.DefaultResourceGroup()),
		KubernetesVersion: pulumi.String(e.KubernetesVersion()),
		AgentPoolProfiles: nodePool,
		LinuxProfile: &containerservice.ContainerServiceLinuxProfileArgs{
			AdminUsername: pulumi.String(adminUsername),
			Ssh: containerservice.ContainerServiceSshConfigurationArgs{
				PublicKeys: containerservice.ContainerServiceSshPublicKeyArray{
					containerservice.ContainerServiceSshPublicKeyArgs{
						KeyData: sshPublicKey,
					},
				},
			},
		},
		AutoUpgradeProfile: containerservice.ManagedClusterAutoUpgradeProfileArgs{
			// Disabling upgrading as this a temporary cluster, we don't want any change after creation
			UpgradeChannel: pulumi.String(containerservice.UpgradeChannelNone),
		},
		DnsPrefix: pulumi.Sprintf("%s-dns", name),
		ApiServerAccessProfile: containerservice.ManagedClusterAPIServerAccessProfileArgs{
			EnablePrivateCluster: pulumi.BoolPtr(false),
		},
		NetworkProfile: containerservice.ContainerServiceNetworkProfileArgs{
			NetworkPlugin: pulumi.String(containerservice.NetworkPluginKubenet),
		},
		Identity: containerservice.ManagedClusterIdentityArgs{
			Type: containerservice.ResourceIdentityTypeSystemAssigned,
		},
		Tags: e.ResourcesTags(),
	}, opts...)
	if err != nil {
		return nil, pulumi.StringOutput{}, err
	}

	creds := containerservice.ListManagedClusterUserCredentialsOutput(e.Ctx,
		containerservice.ListManagedClusterUserCredentialsOutputArgs{
			ResourceGroupName: pulumi.String(e.DefaultResourceGroup()),
			ResourceName:      cluster.Name,
		}, e.WithProvider(config.ProviderAzure),
	)

	kubeconfig := creds.Kubeconfigs().Index(pulumi.Int(0)).Value().
		ApplyT(func(encoded string) string {
			kubeconfig, err := base64.StdEncoding.DecodeString(encoded)
			if err != nil {
				return ""
			}
			return string(kubeconfig)
		}).(pulumi.StringOutput)

	return cluster, kubeconfig, nil
}

func systemNodePool(e azure.Environment, name string) containerservice.ManagedClusterAgentPoolProfileInput {
	return BuildNodePool(e, name, string(containerservice.AgentPoolModeSystem), e.DefaultInstanceType(), string(containerservice.OSTypeLinux), 1)
}

func BuildNodePool(e azure.Environment, name, mode, instanceType, os string, nodeCount int) containerservice.ManagedClusterAgentPoolProfileInput {
	return containerservice.ManagedClusterAgentPoolProfileArgs{
		Name:               pulumi.String(name),
		OsDiskSizeGB:       pulumi.IntPtr(200),
		Count:              pulumi.IntPtr(nodeCount),
		EnableAutoScaling:  pulumi.BoolPtr(false),
		Mode:               pulumi.String(mode),
		EnableNodePublicIP: pulumi.BoolPtr(false),
		Tags:               e.ResourcesTags(),
		OsType:             pulumi.String(os),
		Type:               pulumi.String(containerservice.AgentPoolTypeVirtualMachineScaleSets),
		VmSize:             pulumi.String(instanceType),
	}
}
