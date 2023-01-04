package microVMs

import (
	"crypto/rand"
	"fmt"
	"net"
	"path/filepath"
	"strings"

	_ "embed"

	"github.com/DataDog/test-infra-definitions/aws/scenarios/microVMs/microVMs/resources"
	"github.com/DataDog/test-infra-definitions/aws/scenarios/microVMs/vmconfig"
	"github.com/DataDog/test-infra-definitions/command"
	"github.com/DataDog/test-infra-definitions/common"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi-libvirt/sdk/go/libvirt"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	dhcpEntriesTemplate   = "<host mac='%s' name='%s' ip='%s'/>"
	microVMGroupSubnet    = "169.254.0.0/16"
	domainSocketCreateCmd = `rm -f /tmp/%s.sock && python3 -c "import socket as s; sock = s.socket(s.AF_UNIX); sock.bind('/tmp/%s.sock')"`
)

var subnetGroupMask = net.IPv4Mask(255, 255, 255, 0)

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

func buildDomainSocket(runner *command.Runner, id, resourceName string) (*remote.Command, error) {
	// build domain sockets for fetching logs
	createDomainSocketArgs := command.CommandArgs{
		Create: pulumi.Sprintf(domainSocketCreateCmd, id, id),
	}
	createDomainSocketDone, err := runner.Command(resourceName, &createDomainSocketArgs)
	if err != nil {
		return nil, err
	}

	return createDomainSocketDone, nil
}

type DomainMatrix struct {
	fs          *libvirtFilesystem
	vcpu        int
	memory      int
	xls         string
	domainName  string
	domainID    string
	dhcpEntry   string
	recipe      string
	kernel      *vmconfig.Kernel
	volume      *libvirt.Volume
	domainArgs  *libvirt.DomainArgs
	domainNamer common.Namer
}

func generateNetworkResource(ctx *pulumi.Context, provider *libvirt.Provider, dhcpEntries []string) (*libvirt.Network, error) {

	netXML := fmt.Sprintf(resources.GetRecipeNetworkTemplateOrDefault(""), strings.Join(dhcpEntries[:], ""))
	network, err := libvirt.NewNetwork(ctx, "network", &libvirt.NetworkArgs{
		Addresses: pulumi.StringArray{pulumi.String(microVMGroupSubnet)},
		Mode:      pulumi.String("nat"),
		Xml:       libvirt.NetworkXmlArgs{pulumi.String(netXML)},
	}, pulumi.Provider(provider), pulumi.DeleteBeforeReplace(true))
	if err != nil {
		return nil, err
	}

	return network, nil
}

func libvirtCustomVMArgs(matrix *DomainMatrix) {
	matrix.domainArgs = &libvirt.DomainArgs{
		Consoles: libvirt.DomainConsoleArray{
			libvirt.DomainConsoleArgs{
				Type:       pulumi.String("pty"),
				TargetPort: pulumi.String("0"),
				TargetType: pulumi.String("serial"),
			},
		},
		Disks: libvirt.DomainDiskArray{
			libvirt.DomainDiskArgs{
				VolumeId: matrix.volume.ID(),
			},
		},
		Kernel: pulumi.String(
			filepath.Join(matrix.kernel.Dir, "bzImage"),
		),
		Cmdlines: pulumi.MapArray{
			pulumi.Map{"console": pulumi.String("ttyS0")},
			pulumi.Map{"acpi": pulumi.String("off")},
			pulumi.Map{"panic": pulumi.String("-1")},
			pulumi.Map{"root": pulumi.String("/dev/vda")},
			pulumi.Map{"net.ifnames": pulumi.String("0")},
			pulumi.Map{"_": pulumi.String("rw")},
		},
		Memory: pulumi.Int(matrix.memory),
		Vcpu:   pulumi.Int(matrix.vcpu),
		Xml: libvirt.DomainXmlArgs{
			Xslt: pulumi.String(matrix.xls),
		},
	}
}

func libvirtDistroVMArgs(matrix *DomainMatrix) {
	matrix.domainArgs = &libvirt.DomainArgs{
		Consoles: libvirt.DomainConsoleArray{
			libvirt.DomainConsoleArgs{
				Type:       pulumi.String("pty"),
				TargetPort: pulumi.String("0"),
				TargetType: pulumi.String("serial"),
			},
		},
		Disks: libvirt.DomainDiskArray{
			libvirt.DomainDiskArgs{
				VolumeId: matrix.volume.ID(),
			},
		},
		Memory: pulumi.Int(matrix.memory),
		Vcpu:   pulumi.Int(matrix.vcpu),
		Xml: libvirt.DomainXmlArgs{
			Xslt: pulumi.String(matrix.xls),
		},
	}
}

func setupLibvirtDomainArgs(domainMatrices []*DomainMatrix) error {
	for _, m := range domainMatrices {
		if m.recipe == "custom" {
			libvirtCustomVMArgs(m)
		} else if m.recipe == "distro" {
			libvirtDistroVMArgs(m)
		} else {
			return fmt.Errorf("unknown receipe: %s", m.recipe)
		}
	}

	return nil
}

func newLibvirtFS(ctx *pulumi.Context, vmset *vmconfig.VMSet) (*libvirtFilesystem, error) {
	if vmset.Recipe == "custom" {
		return NewLibvirtFSCustomRecipe(ctx, vmset), nil
	} else if vmset.Recipe == "distro" {
		return NewLibvirtFSDistroRecipe(ctx, vmset), nil
	} else {
		return nil, fmt.Errorf("unknown recipe: %s", vmset.Recipe)
	}
}

func getRecipeDomainXLS(recipe string, args ...any) string {
	template := resources.GetRecipeDomainTemplateOrDefault(recipe)
	if recipe == "custom" {
		return fmt.Sprintf(template, args...)
	}

	return template
}

func buildDomainMatrix(ctx *pulumi.Context, vcpu, memory int, setName, recipe string, kernel vmconfig.Kernel, fs *libvirtFilesystem, ip net.IP) (*DomainMatrix, error) {
	matrix := new(DomainMatrix)
	matrix.domainID = generateDomainIdentifier(vcpu, memory, setName, kernel.Tag)
	matrix.vcpu = vcpu
	matrix.memory = memory
	matrix.domainName = fmt.Sprintf("ddvm-%s", matrix.domainID)

	mac, err := generateNewUnicastMac()
	if err != nil {
		return nil, err
	}

	matrix.dhcpEntry = fmt.Sprintf(dhcpEntriesTemplate, mac, matrix.domainName, ip)
	matrix.xls = getRecipeDomainXLS(recipe, matrix.domainName, sharedFSMountPoint, matrix.domainID, mac)

	matrix.kernel = &kernel
	matrix.recipe = recipe
	matrix.fs = fs
	matrix.domainNamer = common.NewNamer(ctx, matrix.domainID)

	return matrix, nil
}

func buildDomainMatrices(ctx *pulumi.Context, runner *command.Runner, vmsets []vmconfig.VMSet, depends []pulumi.Resource) ([]*DomainMatrix, []pulumi.Resource, error) {
	var matrices []*DomainMatrix
	var waitFor []pulumi.Resource

	ip, _, _ := net.ParseCIDR(microVMGroupSubnet)
	for _, vmset := range vmsets {
		fs, err := newLibvirtFS(ctx, &vmset)
		if err != nil {
			return []*DomainMatrix{}, []pulumi.Resource{}, err
		}
		fsDone, err := fs.setupLibvirtFilesystem(runner, depends)
		if err != nil {
			return []*DomainMatrix{}, []pulumi.Resource{}, err
		}
		waitFor = append(waitFor, fsDone...)

		for _, vcpu := range vmset.VCpu {
			for _, memory := range vmset.Memory {
				for _, kernel := range vmset.Kernels {
					ip = getNextVMSubnet(ip)
					m, err := buildDomainMatrix(ctx, vcpu, memory, vmset.Name, vmset.Recipe, kernel, fs, ip)
					if err != nil {
						return []*DomainMatrix{}, []pulumi.Resource{}, err
					}
					matrices = append(matrices, m)
				}
			}
		}
	}

	return matrices, waitFor, nil
}

func setupDomainVolume(ctx *pulumi.Context, provider *libvirt.Provider, baseVolumeId, poolName, resourceName string) (*libvirt.Volume, error) {
	volume, err := libvirt.NewVolume(ctx, resourceName, &libvirt.VolumeArgs{
		BaseVolumeId: pulumi.String(baseVolumeId),
		Pool:         pulumi.String(poolName),
		Format:       pulumi.String("qcow2"),
	}, pulumi.Provider(provider))
	if err != nil {
		return nil, err
	}

	return volume, nil
}

func setupLibvirtDomainMatrices(ctx *pulumi.Context, runner *command.Runner, libvirtUri pulumi.StringOutput, vmsets []vmconfig.VMSet, depends []pulumi.Resource) ([]*DomainMatrix, *libvirt.Provider, []pulumi.Resource, error) {
	var waitFor []pulumi.Resource
	var matrices []*DomainMatrix

	matrices, waitForMatrices, err := buildDomainMatrices(ctx, runner, vmsets, depends)
	if err != nil {
		return matrices, nil, waitFor, err
	}

	provider, err := libvirt.NewProvider(ctx, "provider", &libvirt.ProviderArgs{
		Uri: libvirtUri,
	}, pulumi.DependsOn(waitForMatrices))
	if err != nil {
		return matrices, nil, waitFor, err
	}

	// setup volumes and domain sockets
	for _, matrix := range matrices {
		baseVolumeId := matrix.fs.baseVolumeMap[matrix.kernel.Tag].volumeKey
		volume, err := setupDomainVolume(ctx, provider, baseVolumeId, matrix.fs.poolName, matrix.domainNamer.ResourceName("volume"))
		if err != nil {
			return matrices, nil, waitFor, err
		}
		matrix.volume = volume

		createDomainSocketDone, err := buildDomainSocket(runner, matrix.domainID, matrix.domainNamer.ResourceName("create-domain-socket"))
		if err != nil {
			return matrices, nil, waitFor, err
		}
		waitFor = append(waitFor, createDomainSocketDone)
	}

	if err := setupLibvirtDomainArgs(matrices); err != nil {
		return matrices, nil, waitFor, err
	}

	return matrices, provider, waitFor, nil
}

func setupLibvirtVMWithRecipe(ctx *pulumi.Context, runner *command.Runner, libvirtUri pulumi.StringOutput, vmsets []vmconfig.VMSet, depends []pulumi.Resource) error {
	var dhcpEntries []string

	matrices, provider, waitFor, err := setupLibvirtDomainMatrices(ctx, runner, libvirtUri, vmsets, depends)
	if err != nil {
		return err
	}

	// build DHCP entries
	for _, m := range matrices {
		dhcpEntries = append(dhcpEntries, m.dhcpEntry)
	}

	network, err := generateNetworkResource(ctx, provider, dhcpEntries)
	if err != nil {
		return err
	}

	// attach network interface to each domain
	for _, matrix := range matrices {
		matrix.domainArgs.NetworkInterfaces = libvirt.DomainNetworkInterfaceArray{
			libvirt.DomainNetworkInterfaceArgs{
				NetworkId:    network.ID(),
				WaitForLease: pulumi.Bool(false),
			},
		}
	}

	for _, matrix := range matrices {
		_, err := libvirt.NewDomain(ctx, matrix.domainNamer.ResourceName("ddvm"), matrix.domainArgs, pulumi.Provider(provider), pulumi.ReplaceOnChanges([]string{"*"}), pulumi.DeleteBeforeReplace(true), pulumi.DependsOn(waitFor))

		if err != nil {
			return err
		}
	}

	return nil
}
