package compute

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/components"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/components/remote"
	"github.com/DataDog/test-infra-definitions/resources/azure"
	"github.com/DataDog/test-infra-definitions/resources/azure/compute"

	network "github.com/pulumi/pulumi-azure-native-sdk/network/v2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// NewVM creates an EC2 Instance and returns a Remote component.
// Without any parameter it creates an Ubuntu VM on AMD64 architecture.
func NewVM(e azure.Environment, name string, params ...VMOption) (*remote.Host, error) {
	vmArgs, err := buildArgs(params...)
	if err != nil {
		return nil, err
	}

	// Default missing parameters
	if err = defaultVMArgs(e, vmArgs); err != nil {
		return nil, err
	}

	// Resolve image URN if necessary
	imageInfo, err := resolveOS(e, *vmArgs)
	if err != nil {
		return nil, err
	}

	// Create the EC2 instance
	return components.NewComponent(&e, e.Namer.ResourceName(name), func(c *remote.Host) error {
		// Create the Azure instance
		var err error
		var nwIface *network.NetworkInterface

		if vmArgs.osInfo.Family() == os.LinuxFamily {
			_, nwIface, err = compute.NewLinuxInstance(e, c.Name(), imageInfo.urn, vmArgs.instanceType, pulumi.StringPtr(vmArgs.userData))
		} else if vmArgs.osInfo.Family() == os.WindowsFamily {
			_, nwIface, _, err = compute.NewWindowsInstance(e, c.Name(), imageInfo.urn, vmArgs.instanceType, pulumi.StringPtr(vmArgs.userData), nil)
		} else {
			return fmt.Errorf("unsupported OS family %v", vmArgs.osInfo.Family())
		}
		if err != nil {
			return err
		}

		// Create connection
		privateIP := nwIface.IpConfigurations.Index(pulumi.Int(0)).PrivateIPAddress()
		conn, err := remote.NewConnection(privateIP.Elem(), compute.AdminUsername, e.DefaultPrivateKeyPath(), e.DefaultPrivateKeyPassword(), "")
		if err != nil {
			return err
		}

		// TODO: Check support of cloud-init on Azure
		return remote.InitHost(&e, conn.ToConnectionOutput(), *vmArgs.osInfo, compute.AdminUsername, command.WaitForSuccessfulConnection, c)
	})
}

func defaultVMArgs(e azure.Environment, vmArgs *vmArgs) error {
	if vmArgs.osInfo == nil {
		vmArgs.osInfo = &os.WindowsDefault
	}

	if vmArgs.instanceType == "" {
		vmArgs.instanceType = e.DefaultInstanceType()
		if vmArgs.osInfo.Architecture == os.ARM64Arch {
			vmArgs.instanceType = e.DefaultARMInstanceType()
		}
	}

	return nil
}
