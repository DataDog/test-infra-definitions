package microVMs

import (
	"fmt"
	"net"
	"os"

	"github.com/DataDog/test-infra-definitions/aws/scenarios/microVMs/vmconfig"
	"github.com/DataDog/test-infra-definitions/command"
	"github.com/pulumi/pulumi-libvirt/sdk/go/libvirt"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const microVMGroupSubnet = "169.254.0.0/16"
const gatewayIP = "169.254.0.0"
const defaultInterface = `
auto eth0
iface eth0 inet static
	address %s
gateway %s
`

var subnetGroupMask = net.IPv4Mask(255, 255, 0, 0)

func setupNetworkInterfaceForVM(runner *command.Runner, fspath pulumi.StringOutput, vmsubnet string, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	setupMountPointArgs := command.CommandArgs{
		Create: pulumi.Sprintf("mkdir -p $HOME/mnt && mount -o exec,loop %s $HOME/mnt", fspath),
		Delete: pulumi.String("rm -r $HOME/mnt"),
		Sudo:   true,
	}
	setupMountPointDone, err := runner.Command("fs-setup-mount", &setupMountPointArgs, pulumi.DependsOn(depends))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	fmt.Printf("%s", fmt.Sprintf(defaultInterface, vmsubnet, gatewayIP))

	writeDefaultInterfaceArgs := command.CommandArgs{
		Create: pulumi.String(
			fmt.Sprintf(defaultInterface, vmsubnet, gatewayIP),
		),
		Sudo: true,
	}
	writeDefaultInterfaceDone, err := runner.Command("fs-write-default-interface", &writeDefaultInterfaceArgs, pulumi.DependsOn([]pulumi.Resource{setupMountPointDone}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	umountFsArgs := command.CommandArgs{
		Create: pulumi.String("umount $HOME/mnt"),
		Sudo:   true,
	}
	umountFsDone, err := runner.Command("fs-umount-volume", &umountFsArgs, pulumi.DependsOn([]pulumi.Resource{writeDefaultInterfaceDone}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	return []pulumi.Resource{umountFsDone}, nil
}

func getNextVMSubnet(ip net.IP) net.IP {
	ipv4 := ip.To4()
	ipv4 = ipv4.Mask(subnetGroupMask)
	ipv4[2]++

	return ipv4
}

func setupLibvirtVM(ctx *pulumi.Context, runner *command.Runner, libvirtUri pulumi.StringOutput, vmset vmconfig.VMSet, waitForList []pulumi.Resource) error {
	// create a provider, this isn't required, but will make it easier to configure
	// a libvirt_uri, which we'll discuss in a bit
	provider, err := libvirt.NewProvider(ctx, "provider", &libvirt.ProviderArgs{
		Uri: libvirtUri,
	}, pulumi.DependsOn(waitForList))
	if err != nil {
		return err
	}

	domainXls, err := os.ReadFile("resources/domain.xls")
	if err != nil {
		return err
	}

	network, err := libvirt.NewNetwork(ctx, "network", &libvirt.NetworkArgs{
		Addresses: pulumi.StringArray{pulumi.String(microVMGroupSubnet)},
		Mode:      pulumi.String("nat"),
	}, pulumi.Provider(provider))
	if err != nil {
		return err
	}

	ip, _, _ := net.ParseCIDR(microVMGroupSubnet)
	for _, kernel := range vmset.Kernels {
		baseVolumeId := generatePoolPath(vmset.Name) + basefsName
		filesystem, err := libvirt.NewVolume(ctx, "filesystem-"+kernel.Tag, &libvirt.VolumeArgs{
			BaseVolumeId: pulumi.String(baseVolumeId),
			Pool:         pulumi.String(vmset.Name),
			Format:       pulumi.String("qcow2"),
		}, pulumi.Provider(provider), pulumi.DependsOn(waitForList))
		if err != nil {
			return err
		}

		done, err := setupNetworkInterfaceForVM(
			runner,
			pulumi.Sprintf("%s/%s", generatePoolPath(vmset.Name), filesystem.Name),
			fmt.Sprintf("%s/24", ip.String()),
			[]pulumi.Resource{filesystem},
		)
		if err != nil {
			return err
		}
		ip = getNextVMSubnet(ip)

		// create a VM that has a name starting with ubuntu
		_, err = libvirt.NewDomain(ctx, "ubuntu", &libvirt.DomainArgs{
			//	Cloudinit: cloud_init.ID(),
			Consoles: libvirt.DomainConsoleArray{
				// enables using `virsh console ...`
				libvirt.DomainConsoleArgs{
					Type:       pulumi.String("pty"),
					TargetPort: pulumi.String("0"),
					TargetType: pulumi.String("serial"),
				},
			},
			Disks: libvirt.DomainDiskArray{
				libvirt.DomainDiskArgs{
					VolumeId: filesystem.ID(),
				},
			},
			NetworkInterfaces: libvirt.DomainNetworkInterfaceArray{
				libvirt.DomainNetworkInterfaceArgs{
					NetworkId:    network.ID(),
					WaitForLease: pulumi.Bool(false),
				},
			},
			Kernel: pulumi.String(kernel.Dir),
			Cmdlines: pulumi.MapArray{
				pulumi.Map{"console": pulumi.String("ttyS0")},
				pulumi.Map{"acpi": pulumi.String("off")},
				pulumi.Map{"panic": pulumi.String("-1")},
				pulumi.Map{"root": pulumi.String("/dev/vda")},
				pulumi.Map{"net.ifnames": pulumi.String("0")},
				pulumi.Map{"_": pulumi.String("rw")},
			},
			Memory: pulumi.Int(4096),
			Vcpu:   pulumi.Int(4),
			Xml: libvirt.DomainXmlArgs{
				Xslt: pulumi.String(string(domainXls)),
			},
			// delete existing VM before creating replacement to avoid two VMs trying to use the same volume
		}, pulumi.Provider(provider), pulumi.ReplaceOnChanges([]string{"*"}), pulumi.DeleteBeforeReplace(true), pulumi.DependsOn(done))
		if err != nil {
			return err
		}
	}

	return nil
}
