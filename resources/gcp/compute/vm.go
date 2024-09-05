package compute

import (
	_ "embed"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/resources/gcp"
	"github.com/pulumi/pulumi-gcp/sdk/v7/go/gcp/compute"
	"github.com/pulumi/pulumi-gcp/sdk/v7/go/gcp/serviceaccount"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func NewLinuxInstance(e gcp.Environment, name string, imageName string, instanceType string, opts ...pulumi.ResourceOption) (*compute.Instance, error) {

	sshPublicKey, err := utils.GetSSHPublicKey(e.DefaultPublicKeyPath())
	if err != nil {
		return nil, err
	}
	_, err = serviceaccount.NewAccount(e.Ctx(), "vm-sa", &serviceaccount.AccountArgs{
		AccountId: pulumi.String("my-vm-sa"),
	}, e.WithProviders(config.ProviderGCP))
	if err != nil {
		return nil, err
	}

	instance, err := compute.NewInstance(e.Ctx(), "vm", &compute.InstanceArgs{
		NetworkInterfaces: compute.InstanceNetworkInterfaceArray{
			&compute.InstanceNetworkInterfaceArgs{
				AccessConfigs: compute.InstanceNetworkInterfaceAccessConfigArray{
					nil,
				},
				Network:    pulumi.String(e.DefaultNetworkName()),
				Subnetwork: pulumi.String(e.DefaultSubnet()),
			},
		},
		Name:        pulumi.String(e.CommonNamer().ResourceName(name)),
		MachineType: pulumi.String(instanceType),
		Tags: pulumi.StringArray{
			pulumi.String("appgate-gateway"),
		},
		BootDisk: &compute.InstanceBootDiskArgs{
			InitializeParams: &compute.InstanceBootDiskInitializeParamsArgs{
				Image: pulumi.String(imageName),
				Labels: pulumi.StringMap{
					"my_label": pulumi.String("value"),
				},
			},
		},
		Metadata: pulumi.StringMap{
			"enable-oslogin": pulumi.String("false"),
			"ssh-keys":       pulumi.Sprintf("gce:%s", sshPublicKey),
		},
		ServiceAccount: &compute.InstanceServiceAccountArgs{
			Scopes: pulumi.StringArray{
				pulumi.String("cloud-platform"),
			},
		},
	}, e.WithProviders(config.ProviderGCP))
	if err != nil {
		return nil, err
	}

	return instance, nil
}
