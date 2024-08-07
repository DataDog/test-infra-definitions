package compute

import (
	_ "embed"
	"github.com/DataDog/test-infra-definitions/resources/gcp"
	"github.com/pulumi/pulumi-gcp/sdk/v7/go/gcp/compute"
	"github.com/pulumi/pulumi-gcp/sdk/v7/go/gcp/serviceaccount"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func NewLinuxInstance(ctx *pulumi.Context, e gcp.Environment, name, userData pulumi.StringPtrInput, opts ...pulumi.ResourceOption) (*compute.Instance, error) {
	_, err := serviceaccount.NewAccount(ctx, "default", &serviceaccount.AccountArgs{
		AccountId: pulumi.String("my-custom-sa"),
	})
	if err != nil {
		return nil, err
	}
	instance, err := compute.NewInstance(ctx, "default", &compute.InstanceArgs{
		NetworkInterfaces: compute.InstanceNetworkInterfaceArray{
			&compute.InstanceNetworkInterfaceArgs{
				AccessConfigs: compute.InstanceNetworkInterfaceAccessConfigArray{
					nil,
				},
				Network: pulumi.String("default"),
			},
		},
		Name:        pulumi.String("my-instance"),
		MachineType: pulumi.String("n2-standard-2"),
		Zone:        pulumi.String("us-central1-a"),
		Tags: pulumi.StringArray{
			pulumi.String("foo"),
			pulumi.String("bar"),
		},
		BootDisk: &compute.InstanceBootDiskArgs{
			InitializeParams: &compute.InstanceBootDiskInitializeParamsArgs{
				Image: pulumi.String("debian-cloud/debian-11"),
				Labels: pulumi.Map{
					"my_label": pulumi.Any("value"),
				},
			},
		},
		ScratchDisks: compute.InstanceScratchDiskArray{
			&compute.InstanceScratchDiskArgs{
				Interface: pulumi.String("NVME"),
			},
		},
		Metadata: pulumi.StringMap{
			"foo": pulumi.String("bar"),
		},
		MetadataStartupScript: pulumi.String("echo hi > /test.txt"),
		ServiceAccount: &compute.InstanceServiceAccountArgs{
			Scopes: pulumi.StringArray{
				pulumi.String("cloud-platform"),
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return instance, nil
}
