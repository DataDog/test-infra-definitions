package aks

import (
	"encoding/base64"
	"math"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/resources/azure"

	"github.com/pulumi/pulumi-azure-native-sdk/containerservice/v2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	adminUsername = "azureuser"

	kataNodePoolName = "kata"

	// Kata runtime constants
	kataSku     = "AzureLinux"
	kataRuntime = "KataMshvVmIsolation"
)

func NewCluster(e azure.Environment, name string, nodePool containerservice.ManagedClusterAgentPoolProfileArray, opts ...pulumi.ResourceOption) (*containerservice.ManagedCluster, pulumi.StringOutput, error) {
	sshPublicKey, err := utils.GetSSHPublicKey(e.DefaultPublicKeyPath())
	if err != nil {
		return nil, pulumi.StringOutput{}, err
	}

	// Warning: we're modifying passed array as it should normally never be used anywhere else
	nodePool = append(nodePool, systemNodePool(e, "system"))

	if e.DeployKata() {
		nodePool = append(nodePool, kataNodePool(e))
	}

	opts = append(opts, e.WithProviders(config.ProviderAzure))
	cluster, err := containerservice.NewManagedCluster(e.Ctx, e.Namer.ResourceName(name), &containerservice.ManagedClusterArgs{
		ResourceName:      e.CommonNamer.DisplayName(math.MaxInt, pulumi.String(name)),
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
	return BuildNodePool(NodePoolParams{
		Environment:  e,
		Name:         name,
		Mode:         string(containerservice.AgentPoolModeSystem),
		InstanceType: e.DefaultInstanceType(),
		OSType:       string(containerservice.OSTypeLinux),
		NodeCount:    1,
	})
}

func kataNodePool(e azure.Environment) containerservice.ManagedClusterAgentPoolProfileInput {
	return BuildNodePool(NodePoolParams{
		Environment:     e,
		Name:            kataNodePoolName,
		Mode:            string(containerservice.AgentPoolModeSystem),
		InstanceType:    e.KataInstanceType(),
		OSType:          string(containerservice.OSTypeLinux),
		NodeCount:       2,
		WorkloadRuntime: kataRuntime,
		OsSku:           kataSku,
	})
}

type NodePoolParams struct {
	Environment     azure.Environment
	Name            string
	Mode            string
	InstanceType    string
	OSType          string
	OsSku           string
	NodeCount       int
	WorkloadRuntime string
}

func BuildNodePool(params NodePoolParams) containerservice.ManagedClusterAgentPoolProfileInput {
	e := params.Environment
	return containerservice.ManagedClusterAgentPoolProfileArgs{
		Name:               pulumi.String(params.Name),
		OsDiskSizeGB:       pulumi.IntPtr(200),
		Count:              pulumi.IntPtr(params.NodeCount),
		EnableAutoScaling:  pulumi.BoolPtr(false),
		Mode:               pulumi.String(params.Mode),
		EnableNodePublicIP: pulumi.BoolPtr(false),
		Tags:               e.ResourcesTags(),
		OsType:             pulumi.String(params.OSType),
		Type:               pulumi.String(containerservice.AgentPoolTypeVirtualMachineScaleSets),
		VmSize:             pulumi.String(params.InstanceType),
		WorkloadRuntime:    pulumi.String(params.WorkloadRuntime),
		OsSKU:              pulumi.String(params.OsSku),
	}
}
