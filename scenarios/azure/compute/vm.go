package compute

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/components"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/components/remote"
	"github.com/DataDog/test-infra-definitions/resources/azure"
	"github.com/DataDog/test-infra-definitions/resources/azure/compute"

	azNetwork "github.com/pulumi/pulumi-azure-native-sdk/network"
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
	if err = defaultVMArgs(vmArgs); err != nil {
		return nil, err
	}

	// Resolve image URN if necessary
	imageInfo, err := resolveOS(e, *vmArgs)
	if err != nil {
		return nil, err
	}

	// Create the EC2 instance
	return components.NewComponent(*e.CommonEnvironment, e.Namer.ResourceName(name), func(c *remote.Host) error {
		// Create the Azure instance
		var err error
		var nwIface *azNetwork.NetworkInterface

		if vmArgs.osInfo.Family() == os.LinuxFamily {
			_, _, nwIface, err = compute.NewLinuxInstance(e, c.Name(), imageInfo.urn, vmArgs.instanceType, pulumi.StringPtr(vmArgs.userData))
		} else if vmArgs.osInfo.Family() == os.WindowsFamily {
			_, _, nwIface, _, err = compute.NewWindowsInstance(e, c.Name(), imageInfo.urn, vmArgs.instanceType, pulumi.StringPtr(vmArgs.userData), nil)
		} else {
			return fmt.Errorf("unsupported OS family %v", vmArgs.osInfo.Family())
		}
		if err != nil {
			return err
		}

		// Create connection
		privateIP := nwIface.IpConfigurations.Index(pulumi.Int(0)).PrivateIPAddress()
		conn, err := remote.MakeConnection(privateIP.Elem(), compute.AdminUsername, e.DefaultPrivateKeyPath(), e.DefaultPrivateKeyPassword(), "")
		if err != nil {
			return err
		}

		// TODO: Check support of cloud-init on Azure
		return remote.MakeHost(*e.CommonEnvironment, conn.ToConnectionOutput(), *vmArgs.osInfo, compute.AdminUsername, command.WaitUntilSuccess, c)
	})
}

func defaultVMArgs(vmArgs *vmArgs) error {
	if vmArgs.osInfo == nil {
		vmArgs.osInfo = &os.UbuntuDefault
	}

	return nil
}
