package microVMs

import (
	"crypto/rand"
	"fmt"
	"net"
	"path/filepath"
	"strings"

	_ "embed"

	"github.com/DataDog/test-infra-definitions/aws/scenarios/microVMs/vmconfig"
	"github.com/DataDog/test-infra-definitions/command"
	"github.com/pulumi/pulumi-libvirt/sdk/go/libvirt"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	dhcpEntriesTemplate   = "<host mac='%s' name='%s' ip='%s'/>"
	microVMGroupSubnet    = "169.254.0.0/16"
	domainSocketCreateCmd = `rm -f /tmp/%s.sock && python3 -c "import socket as s; sock = s.socket(s.AF_UNIX); sock.bind('/tmp/%s.sock')"`
)

//go:embed resources/domain.xls
var domainXLS string

//go:embed resources/network.xls
var netXMLTemplate string

var subnetGroupMask = net.IPv4Mask(255, 255, 255, 0)

type DomainTemplate struct {
	Mac string
}

type DomainParameters struct {
	filesystemID pulumi.IDOutput
	ip           string
	xls          string
	domainName   string
}

func getNextVMSubnet(ip net.IP) net.IP {
	ipv4 := ip.To4()
	ipv4 = ipv4.Mask(subnetGroupMask)
	ipv4[2]++

	return ipv4
}

func generateNewUnicastMac() (string, error) {
	buf := make([]byte, 6)
	_, err := rand.Read(buf)
	if err != nil {
		return "", err
	}

	// Set LSB bit of MSB byte to 0
	// This denotes unicast mac address
	buf[0] &= 0xfe
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", buf[0], buf[1], buf[2], buf[3], buf[4], buf[5]), nil
}

func setupLibvirtVM(ctx *pulumi.Context, runner *command.Runner, libvirtUri pulumi.StringOutput, vmset vmconfig.VMSet, waitForList []pulumi.Resource) error {
	var domainDependencies []pulumi.Resource
	baseVolumeId := generatePoolPath(vmset.Name) + basefsName

	provider, err := libvirt.NewProvider(ctx, "provider", &libvirt.ProviderArgs{
		Uri: libvirtUri,
	}, pulumi.DependsOn(waitForList))
	if err != nil {
		return err
	}

	var dhcpEntries []string
	domainParameters := make(map[string]*DomainParameters)
	ip, _, _ := net.ParseCIDR(microVMGroupSubnet)
	for _, kernel := range vmset.Kernels {
		params := DomainParameters{}

		if _, ok := domainParameters[kernel.Tag]; ok {
			return fmt.Errorf("duplicate kernel tag: %s", kernel.Tag)
		}

		// Use base volume to generate new filesystem
		filesystem, err := libvirt.NewVolume(ctx, "filesystem-"+kernel.Tag, &libvirt.VolumeArgs{
			BaseVolumeId: pulumi.String(baseVolumeId),
			Pool:         pulumi.String(vmset.Name),
			Format:       pulumi.String("qcow2"),
		}, pulumi.Provider(provider), pulumi.DependsOn(waitForList))
		if err != nil {
			return err
		}

		ip = getNextVMSubnet(ip)

		params.filesystemID = filesystem.ID()
		params.ip = ip.String()
		mac, err := generateNewUnicastMac()
		if err != nil {
			return err
		}

		// build domain sockets for fetching logs
		createDomainSocketArgs := command.CommandArgs{
			Create: pulumi.String(
				fmt.Sprintf(domainSocketCreateCmd, kernel.Tag, kernel.Tag),
			),
		}
		createDomainSocketDone, err := runner.Command("create-domain-socket-"+kernel.Tag, &createDomainSocketArgs)
		if err != nil {
			return err
		}
		domainDependencies = append(domainDependencies, createDomainSocketDone)

		params.domainName = fmt.Sprintf("ddvm-custom-%s", kernel.Tag)
		params.xls = fmt.Sprintf(string(domainXLS), params.domainName, sharedFSMountPoint, kernel.Tag, mac)
		domainParameters[kernel.Tag] = &params

		dhcpEntries = append(dhcpEntries, fmt.Sprintf(dhcpEntriesTemplate, mac, params.domainName, params.ip))

	}

	netXML := fmt.Sprintf(string(netXMLTemplate), strings.Join(dhcpEntries[:], ""))
	network, err := libvirt.NewNetwork(ctx, "network", &libvirt.NetworkArgs{
		Addresses: pulumi.StringArray{pulumi.String(microVMGroupSubnet)},
		Mode:      pulumi.String("nat"),
		Xml:       libvirt.NetworkXmlArgs{pulumi.String(netXML)},
	}, pulumi.Provider(provider), pulumi.DeleteBeforeReplace(true))
	if err != nil {
		return err
	}

	for _, kernel := range vmset.Kernels {
		domainParams, ok := domainParameters[kernel.Tag]
		if !ok {
			return fmt.Errorf("missing parameters for domain %s", kernel.Tag)
		}

		_, err = libvirt.NewDomain(ctx, "ubuntu"+kernel.Tag, &libvirt.DomainArgs{
			Consoles: libvirt.DomainConsoleArray{
				libvirt.DomainConsoleArgs{
					Type:       pulumi.String("pty"),
					TargetPort: pulumi.String("0"),
					TargetType: pulumi.String("serial"),
				},
			},
			Disks: libvirt.DomainDiskArray{
				libvirt.DomainDiskArgs{
					VolumeId: domainParams.filesystemID,
				},
			},
			NetworkInterfaces: libvirt.DomainNetworkInterfaceArray{
				libvirt.DomainNetworkInterfaceArgs{
					NetworkId:    network.ID(),
					WaitForLease: pulumi.Bool(false),
				},
			},
			Kernel: pulumi.String(
				filepath.Join(kernel.Dir, "bzImage"),
			),
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
				Xslt: pulumi.String(domainParams.xls),
			},
			// delete existing VM before creating replacement to avoid two VMs trying to use the same volume
		}, pulumi.Provider(provider), pulumi.ReplaceOnChanges([]string{"*"}), pulumi.DeleteBeforeReplace(true), pulumi.DependsOn(domainDependencies))
		if err != nil {
			return err
		}
	}

	return nil
}
