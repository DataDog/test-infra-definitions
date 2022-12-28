package microVMs

import (
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"strings"

	_ "embed"

	"github.com/DataDog/test-infra-definitions/aws/scenarios/microVMs/vmconfig"
	"github.com/DataDog/test-infra-definitions/command"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi-libvirt/sdk/go/libvirt"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	dhcpEntriesTemplate      = "<host mac='%s' name='%s' ip='%s'/>"
	microVMCustomGroupSubnet = "169.254.0.0/16"
	microVMDistroGroupSubnet = "179.254.0.0/16"
	domainSocketCreateCmd    = `rm -f /tmp/%s.sock && python3 -c "import socket as s; sock = s.socket(s.AF_UNIX); sock.bind('/tmp/%s.sock')"`
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
	vcpu         int
	memory       int
	dhcpEntry    string
	kernel       vmconfig.Kernel
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

func generateDomainIdentifier(vcpu, memory int, vmsetName, tag string) string {
	return fmt.Sprintf("%s-tag-%s-cpu-%d-mem-%d", vmsetName, tag, vcpu, memory)
}

func buildDomainSocket(runner *command.Runner, id string) (*remote.Command, error) {
	// build domain sockets for fetching logs
	createDomainSocketArgs := command.CommandArgs{
		Create: pulumi.String(
			fmt.Sprintf(domainSocketCreateCmd, id, id),
		),
	}
	createDomainSocketDone, err := runner.Command("create-domain-socket-"+id, &createDomainSocketArgs)
	if err != nil {
		return nil, err
	}

	return createDomainSocketDone, nil
}

func buildDomainParameters(ctx *pulumi.Context, provider *libvirt.Provider, runner *command.Runner, vmset *vmconfig.VMSet) (map[string]*DomainParameters, []pulumi.Resource, error) {
	baseVolumeId := generatePoolPath(vmset.Name) + basefsName
	domainParameters := make(map[string]*DomainParameters)
	var domainDependencies []pulumi.Resource

	if len(vmset.Kernels)*len(vmset.Memory)*len(vmset.VCpu) > 255 {
		return domainParameters, []pulumi.Resource{}, errors.New("matrix of length greater than 255 not currently supported")
	}
	ip, _, _ := net.ParseCIDR(microVMCustomGroupSubnet)

	for _, vcpu := range vmset.VCpu {
		for _, memory := range vmset.Memory {
			for _, kernel := range vmset.Kernels {
				id := generateDomainIdentifier(vcpu, memory, vmset.Name, kernel.Tag)
				params := DomainParameters{}

				if _, ok := domainParameters[id]; ok {
					return domainParameters, []pulumi.Resource{}, fmt.Errorf("duplicate domain: %s", id)
				}

				// Use base volume to generate new filesystem
				filesystem, err := libvirt.NewVolume(ctx, "filesystem-"+id, &libvirt.VolumeArgs{
					BaseVolumeId: pulumi.String(baseVolumeId),
					Pool:         pulumi.String(vmset.Name),
					Format:       pulumi.String("qcow2"),
				}, pulumi.Provider(provider))
				if err != nil {
					return domainParameters, []pulumi.Resource{}, err
				}

				ip = getNextVMSubnet(ip)

				params.filesystemID = filesystem.ID()
				params.ip = ip.String()
				mac, err := generateNewUnicastMac()
				if err != nil {
					return domainParameters, []pulumi.Resource{}, err
				}

				createDomainSocketDone, err := buildDomainSocket(runner, id)
				if err != nil {
					return domainParameters, []pulumi.Resource{}, err
				}
				domainDependencies = append(domainDependencies, createDomainSocketDone)

				params.domainName = fmt.Sprintf("ddvm-%s", id)
				params.xls = fmt.Sprintf(string(domainXLS), params.domainName, sharedFSMountPoint, id, mac)
				params.dhcpEntry = fmt.Sprintf(dhcpEntriesTemplate, mac, params.domainName, params.ip)
				params.kernel = kernel
				params.vcpu = vcpu
				params.memory = memory
				domainParameters[id] = &params
			}
		}
	}

	return domainParameters, domainDependencies, nil
}

func generateNetworkResource(ctx *pulumi.Context, provider *libvirt.Provider, domainParameters map[string]*DomainParameters) (*libvirt.Network, error) {
	var dhcpEntries []string
	for _, params := range domainParameters {
		dhcpEntries = append(dhcpEntries, params.dhcpEntry)
	}

	netXML := fmt.Sprintf(string(netXMLTemplate), strings.Join(dhcpEntries[:], ""))
	network, err := libvirt.NewNetwork(ctx, "network", &libvirt.NetworkArgs{
		Addresses: pulumi.StringArray{pulumi.String(microVMCustomGroupSubnet)},
		Mode:      pulumi.String("nat"),
		Xml:       libvirt.NetworkXmlArgs{pulumi.String(netXML)},
	}, pulumi.Provider(provider), pulumi.DeleteBeforeReplace(true))
	if err != nil {
		return nil, err
	}

	return network, nil
}

func libvirtCustomVMArgs(domainParameters map[string]*DomainParameters, network *libvirt.Network) map[string]*libvirt.DomainArgs {
	domainArgs := make(map[string]*libvirt.DomainArgs)
	for domainID, params := range domainParameters {
		domainArgs[domainID] = &libvirt.DomainArgs{
			Consoles: libvirt.DomainConsoleArray{
				libvirt.DomainConsoleArgs{
					Type:       pulumi.String("pty"),
					TargetPort: pulumi.String("0"),
					TargetType: pulumi.String("serial"),
				},
			},
			Disks: libvirt.DomainDiskArray{
				libvirt.DomainDiskArgs{
					VolumeId: params.filesystemID,
				},
			},
			NetworkInterfaces: libvirt.DomainNetworkInterfaceArray{
				libvirt.DomainNetworkInterfaceArgs{
					NetworkId:    network.ID(),
					WaitForLease: pulumi.Bool(false),
				},
			},
			Kernel: pulumi.String(
				filepath.Join(params.kernel.Dir, "bzImage"),
			),
			Cmdlines: pulumi.MapArray{
				pulumi.Map{"console": pulumi.String("ttyS0")},
				pulumi.Map{"acpi": pulumi.String("off")},
				pulumi.Map{"panic": pulumi.String("-1")},
				pulumi.Map{"root": pulumi.String("/dev/vda")},
				pulumi.Map{"net.ifnames": pulumi.String("0")},
				pulumi.Map{"_": pulumi.String("rw")},
			},
			Memory: pulumi.Int(params.memory),
			Vcpu:   pulumi.Int(params.vcpu),
			Xml: libvirt.DomainXmlArgs{
				Xslt: pulumi.String(params.xls),
			},
		}
	}

	return domainArgs
}

func libvirtDistroVMArgs(domainParameters map[string]*DomainParameters, network *libvirt.Network) map[string]*libvirt.DomainArgs {
	domainArgs := make(map[string]*libvirt.DomainArgs)
	for domainID, params := range domainParameters {
		domainArgs[domainID] = &libvirt.DomainArgs{
			Consoles: libvirt.DomainConsoleArray{
				libvirt.DomainConsoleArgs{
					Type:       pulumi.String("pty"),
					TargetPort: pulumi.String("0"),
					TargetType: pulumi.String("serial"),
				},
			},
			Disks: libvirt.DomainDiskArray{
				libvirt.DomainDiskArgs{
					VolumeId: params.filesystemID,
				},
			},
			NetworkInterfaces: libvirt.DomainNetworkInterfaceArray{
				libvirt.DomainNetworkInterfaceArgs{
					NetworkId:    network.ID(),
					WaitForLease: pulumi.Bool(false),
				},
			},
			Memory: pulumi.Int(params.memory),
			Vcpu:   pulumi.Int(params.vcpu),
			Xml: libvirt.DomainXmlArgs{
				Xslt: pulumi.String(params.xls),
			},
		}
	}

	return domainArgs
}

func setupLibvirtVMWithRecipe(ctx *pulumi.Context, runner *command.Runner, libvirtUri pulumi.StringOutput, vmset *vmconfig.VMSet, depends []pulumi.Resource) error {
	var domainArgs map[string]*libvirt.DomainArgs

	fs := NewLibvirtFS(vmset.Name, &vmset.Img)
	fsDone, err := fs.setupLibvirtFilesystem(runner, depends)
	if err != nil {
		return err
	}

	provider, err := libvirt.NewProvider(ctx, "provider", &libvirt.ProviderArgs{
		Uri: libvirtUri,
	}, pulumi.DependsOn(fsDone))
	if err != nil {
		return err
	}

	domainParameters, domainDependencies, err := buildDomainParameters(ctx, provider, runner, vmset)
	if err != nil {
		return err
	}

	network, err := generateNetworkResource(ctx, provider, domainParameters)
	if err != nil {
		return err
	}

	if vmset.Recipe == "custom" {
		domainArgs = libvirtCustomVMArgs(domainParameters, network)
		//	} else if vmset.Recipe == "distro" {
		//		domainArgs = libvirtDistroVMArgs(domainParameters, network)
	} else {
		return fmt.Errorf("unknown receipe: %s", vmset.Recipe)
	}

	for domainID, arg := range domainArgs {
		_, err := libvirt.NewDomain(ctx, "dd-vm-"+domainID, arg, pulumi.Provider(provider), pulumi.ReplaceOnChanges([]string{"*"}), pulumi.DeleteBeforeReplace(true), pulumi.DependsOn(domainDependencies))

		if err != nil {
			return err
		}
	}

	return nil
}
