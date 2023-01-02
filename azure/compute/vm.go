package compute

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/azure"
	"github.com/DataDog/test-infra-definitions/common/utils"

	"github.com/pulumi/pulumi-azure-native-sdk/compute"
	"github.com/pulumi/pulumi-azure-native-sdk/network"
	"github.com/pulumi/pulumi-random/sdk/v4/go/random"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	imageURNSeparator = ":"
	adminUsername     = "azureuser"
)

func NewLinuxInstance(e azure.Environment, name, imageUrn, instanceType string, userData pulumi.StringPtrInput) (*compute.VirtualMachine, *network.PublicIPAddress, *network.NetworkInterface, error) {
	sshPublicKey, err := utils.GetSSHPublicKey(e.DefaultPublicKeyPath())
	if err != nil {
		return nil, nil, nil, err
	}

	linuxOsProfile := compute.OSProfileArgs{
		ComputerName:  pulumi.String(name),
		AdminUsername: pulumi.String(adminUsername),
		LinuxConfiguration: compute.LinuxConfigurationArgs{
			DisablePasswordAuthentication: pulumi.BoolPtr(true),
			Ssh: compute.SshConfigurationArgs{
				PublicKeys: compute.SshPublicKeyTypeArray{
					compute.SshPublicKeyTypeArgs{
						KeyData: sshPublicKey,
						Path:    pulumi.StringPtr(fmt.Sprintf("/home/%s/.ssh/authorized_keys", adminUsername)),
					},
				},
			},
		},
		CustomData: userData,
	}

	return newVMInstance(e, name, imageUrn, instanceType, linuxOsProfile, nil)
}

func NewWindowsInstance(e azure.Environment, name, imageUrn, instanceType string, userData, firstLogonCommand pulumi.StringPtrInput) (*compute.VirtualMachine, *network.PublicIPAddress, *network.NetworkInterface, pulumi.StringOutput, error) {
	windowsAdminPassword, err := random.NewRandomPassword(e.Ctx, e.Namer.ResourceName(name, "admin-password"), &random.RandomPasswordArgs{
		Length:  pulumi.Int(20),
		Special: pulumi.Bool(true),
	})
	if err != nil {
		return nil, nil, nil, pulumi.StringOutput{}, err
	}

	windowsOsProfile := compute.OSProfileArgs{
		ComputerName:  pulumi.String(name),
		AdminUsername: pulumi.String(adminUsername),
		AdminPassword: windowsAdminPassword.Result,
		CustomData:    userData,
	}

	if firstLogonCommand != nil {
		windowsOsProfile.WindowsConfiguration = compute.WindowsConfigurationArgs{
			AdditionalUnattendContent: compute.AdditionalUnattendContentArray{
				compute.AdditionalUnattendContentArgs{
					ComponentName: compute.ComponentNames_Microsoft_Windows_Shell_Setup,
					PassName:      compute.PassNamesOobeSystem,
					SettingName:   compute.SettingNamesFirstLogonCommands,
					Content:       firstLogonCommand,
				},
			},
		}
	}

	securityGroup, err := network.NewNetworkSecurityGroup(e.Ctx, e.Namer.ResourceName(name, "security-group-rdp"), &network.NetworkSecurityGroupArgs{
		ResourceGroupName: pulumi.String(e.DefaultResourceGroup()),
		SecurityRules: network.SecurityRuleTypeArray{
			network.SecurityRuleTypeArgs{
				Access:                   pulumi.String("Allow"),
				DestinationAddressPrefix: pulumi.String("*"),
				Direction:                pulumi.String("Inbound"),
				DestinationPortRange:     pulumi.String("3389"), // RDP
				Protocol:                 pulumi.String("TCP"),
				SourceAddressPrefix:      pulumi.String("*"),
				SourcePortRange:          pulumi.String("*"),
				Name:                     pulumi.String(e.Namer.ResourceName(name, "security-rule-rdp")),
				Priority:                 pulumi.Int(100),
			},
		},
	})
	if err != nil {
		return nil, nil, nil, pulumi.StringOutput{}, err
	}

	vm, publicIP, nw, err := newVMInstance(e, name, imageUrn, instanceType, windowsOsProfile, securityGroup)
	if err != nil {
		return nil, nil, nil, pulumi.StringOutput{}, err
	}

	return vm, publicIP, nw, windowsAdminPassword.Result, nil
}

func newVMInstance(
	e azure.Environment,
	name, imageUrn, instanceType string,
	osProfile compute.OSProfilePtrInput,
	securityGroup *network.NetworkSecurityGroup) (*compute.VirtualMachine, *network.PublicIPAddress, *network.NetworkInterface, error) {
	vmImageRef, err := parseImageReferenceURN(imageUrn)
	if err != nil {
		return nil, nil, nil, err
	}

	publicIP, err := network.NewPublicIPAddress(e.Ctx, e.Namer.ResourceName(name), &network.PublicIPAddressArgs{
		PublicIpAddressName:      e.Namer.DisplayName(pulumi.String(name)),
		ResourceGroupName:        pulumi.String(e.DefaultResourceGroup()),
		PublicIPAllocationMethod: pulumi.String(network.IPAllocationMethodStatic),
		Tags:                     e.ResourcesTags(),
	}, pulumi.Provider(e.Provider))
	if err != nil {
		return nil, nil, nil, err
	}

	networkArgs := &network.NetworkInterfaceArgs{
		NetworkInterfaceName: e.Namer.DisplayName(pulumi.String(name)),
		ResourceGroupName:    pulumi.String(e.DefaultResourceGroup()),
		IpConfigurations: network.NetworkInterfaceIPConfigurationArray{
			network.NetworkInterfaceIPConfigurationArgs{
				Name: e.Namer.DisplayName(pulumi.String(name)),
				Subnet: network.SubnetTypeArgs{
					Id: pulumi.String(e.DefaultSubnet()),
				},
				PrivateIPAllocationMethod: pulumi.String(network.IPAllocationMethodDynamic),
				PublicIPAddress: network.PublicIPAddressTypeArgs{
					Id: publicIP.ID(),
				},
			},
		},
		Tags: e.ResourcesTags(),
	}

	if securityGroup != nil {
		networkArgs.NetworkSecurityGroup = network.NetworkSecurityGroupTypeArgs{
			Id: securityGroup.ID(),
		}
	}

	nwInt, err := network.NewNetworkInterface(e.Ctx, e.Namer.ResourceName(name), networkArgs, pulumi.Provider(e.Provider))
	if err != nil {
		return nil, nil, nil, err
	}

	vm, err := compute.NewVirtualMachine(e.Ctx, e.Namer.ResourceName(name), &compute.VirtualMachineArgs{
		ResourceGroupName: pulumi.String(e.DefaultResourceGroup()),
		VmName:            e.Namer.DisplayName(pulumi.String(name)),
		HardwareProfile: compute.HardwareProfileArgs{
			VmSize: pulumi.StringPtr(instanceType),
		},
		StorageProfile: compute.StorageProfileArgs{
			OsDisk: compute.OSDiskArgs{
				Name:         e.Namer.DisplayName(pulumi.String(name), pulumi.String("os-disk")),
				CreateOption: pulumi.String(compute.DiskCreateOptionFromImage),
				ManagedDisk: compute.ManagedDiskParametersArgs{
					StorageAccountType: pulumi.String("StandardSSD_LRS"),
				},
				DeleteOption: pulumi.String(compute.DiskDeleteOptionTypesDelete),
				DiskSizeGB:   pulumi.IntPtr(200), // Windows requires at least 127GB
			},
			ImageReference: vmImageRef,
		},
		NetworkProfile: compute.NetworkProfileArgs{
			NetworkInterfaces: compute.NetworkInterfaceReferenceArray{
				compute.NetworkInterfaceReferenceArgs{
					Id: nwInt.ID(),
				},
			},
		},
		OsProfile: osProfile,
		Tags:      e.ResourcesTags(),
	}, pulumi.Provider(e.Provider))
	if err != nil {
		return nil, nil, nil, err
	}

	return vm, publicIP, nwInt, nil
}
