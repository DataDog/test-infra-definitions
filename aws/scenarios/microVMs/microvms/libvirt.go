package microvms

import (
	"fmt"
	"net"
	"path/filepath"
	"strings"

	"github.com/DataDog/test-infra-definitions/aws/scenarios/microVMs/microvms/resources"
	"github.com/DataDog/test-infra-definitions/aws/scenarios/microVMs/vmconfig"
	"github.com/DataDog/test-infra-definitions/command"
	"github.com/DataDog/test-infra-definitions/common/namer"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi-libvirt/sdk/go/libvirt"
	"github.com/pulumi/pulumi-random/sdk/v4/go/random"
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

func generateNewUnicastMac(ctx *pulumi.Context, domainID string) (pulumi.StringOutput, error) {
	pulumiRandStr, err := random.NewRandomString(ctx, "random-"+domainID, &random.RandomStringArgs{
		Length:  pulumi.Int(6),
		Special: pulumi.Bool(true),
	})
	if err != nil {
		return pulumi.StringOutput{}, err
	}

	macAddr := pulumiRandStr.Result.ApplyT(func(randStr string) string {
		buf := []byte(randStr)

		// Set LSB bit of MSB byte to 0
		// This denotes unicast mac address
		buf[0] &= 0xfe

		return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", buf[0], buf[1], buf[2], buf[3], buf[4], buf[5])
	}).(pulumi.StringOutput)

	return macAddr, nil
}

func generateDomainIdentifier(vcpu, memory int, vmsetName, tag string) string {
	return fmt.Sprintf("%s-tag-%s-cpu-%d-mem-%d", vmsetName, tag, vcpu, memory)
}

func buildDomainSocket(runner *command.Runner, id, resourceName string, depends []pulumi.Resource) (*remote.Command, error) {
	// build domain sockets for fetching logs
	createDomainSocketArgs := command.Args{
		Create: pulumi.Sprintf(domainSocketCreateCmd, id, id),
	}
	createDomainSocketDone, err := runner.Command(resourceName, &createDomainSocketArgs, pulumi.DependsOn(depends))
	if err != nil {
		return nil, err
	}

	return createDomainSocketDone, nil
}

type DomainMatrix struct {
	resources.RecipeLibvirtDomainArgs
	fs          *LibvirtFilesystem
	domainName  string
	domainID    string
	dhcpEntry   pulumi.StringOutput
	arch        string
	kernel      *vmconfig.Kernel
	domainArgs  *libvirt.DomainArgs
	domainNamer namer.Namer
	instance    *Instance
}

func generateNetworkResource(ctx *pulumi.Context, provider *libvirt.Provider, resourceNamer namer.Namer, dhcpEntries []interface{}) (*libvirt.Network, error) {

	// Collect all DHCP entries in a single string to be
	// formatted in network XML.
	dhcpEntriesJoined := pulumi.All(dhcpEntries...).ApplyT(
		func(promises []interface{}) (string, error) {
			var sb strings.Builder

			for _, promise := range promises {
				sb.WriteString(promise.(string))
			}

			return sb.String(), nil
		},
	).(pulumi.StringInput)

	netXML := resources.GetDefaultNetworkXLS(
		map[string]pulumi.StringInput{
			resources.DHCPEntries: dhcpEntriesJoined,
		},
	)
	network, err := libvirt.NewNetwork(ctx, resourceNamer.ResourceName("network"), &libvirt.NetworkArgs{
		Addresses: pulumi.StringArray{pulumi.String(microVMGroupSubnet)},
		Mode:      pulumi.String("nat"),
		Xml: libvirt.NetworkXmlArgs{
			Xslt: netXML,
		},
	}, pulumi.Provider(provider), pulumi.DeleteBeforeReplace(true))
	if err != nil {
		return nil, err
	}

	return network, nil
}

func newLibvirtFS(ctx *pulumi.Context, vmset *vmconfig.VMSet) (*LibvirtFilesystem, error) {
	if vmset.Recipe == "custom-arm64" {
		return NewLibvirtFSCustomRecipe(ctx, vmset), nil
	} else if vmset.Recipe == "custom-amd64" {
		return NewLibvirtFSCustomRecipe(ctx, vmset), nil
	} else if vmset.Recipe == "distro" {
		return NewLibvirtFSDistroRecipe(ctx, vmset), nil
	} else {
		return nil, fmt.Errorf("unknown recipe: %s", vmset.Recipe)
	}
}

func buildDomainMatrix(ctx *pulumi.Context, vcpu, memory int, setName string, rc resources.ResourceCollection, instance *Instance, kernel vmconfig.Kernel, fs *LibvirtFilesystem, ip net.IP) (*DomainMatrix, error) {
	matrix := new(DomainMatrix)
	matrix.domainID = generateDomainIdentifier(vcpu, memory, setName, kernel.Tag)
	matrix.arch = instance.Arch
	matrix.instance = instance
	matrix.domainName = fmt.Sprintf("ddvm-%s", matrix.domainID)

	mac, err := generateNewUnicastMac(ctx, matrix.domainID)
	if err != nil {
		return nil, err
	}

	matrix.dhcpEntry = pulumi.Sprintf(dhcpEntriesTemplate, mac, matrix.domainName, ip)
	matrix.kernel = &kernel
	matrix.fs = fs
	matrix.domainNamer = namer.NewNamer(ctx, matrix.domainID)

	matrix.RecipeLibvirtDomainArgs.Vcpu = vcpu
	matrix.RecipeLibvirtDomainArgs.Memory = memory
	matrix.RecipeLibvirtDomainArgs.KernelPath = filepath.Join(kernel.Dir, "bzImage")
	matrix.RecipeLibvirtDomainArgs.Xls = rc.GetDomainXLS(
		map[string]pulumi.StringInput{
			resources.DomainName:    pulumi.String(matrix.domainName),
			resources.SharedFSMount: pulumi.String(sharedFSMountPoint),
			resources.DomainID:      pulumi.String(matrix.domainID),
			resources.MACAddress:    mac,
		},
	)
	matrix.RecipeLibvirtDomainArgs.Resources = rc
	matrix.RecipeLibvirtDomainArgs.ExtraKernelParams = kernel.ExtraParams

	return matrix, nil
}

func buildDomainMatrices(instances map[string]*Instance, vmsets []vmconfig.VMSet, depends []pulumi.Resource) ([]*DomainMatrix, []pulumi.Resource, error) {
	var matrices []*DomainMatrix
	var waitFor []pulumi.Resource

	ip, _, _ := net.ParseCIDR(microVMGroupSubnet)
	for _, vmset := range vmsets {
		rc := resources.NewResourceCollection(vmset.Recipe)
		instance, ok := instances[vmset.Arch]
		if !ok {
			return []*DomainMatrix{}, []pulumi.Resource{}, fmt.Errorf("unsupported arch: %s", vmset.Arch)
		}

		fs, err := newLibvirtFS(instance.ctx, &vmset)
		if err != nil {
			return []*DomainMatrix{}, []pulumi.Resource{}, err
		}
		fsDone, err := fs.setupLibvirtFilesystem(instance.remoteRunner, depends)
		if err != nil {
			return []*DomainMatrix{}, []pulumi.Resource{}, err
		}
		waitFor = append(waitFor, fsDone...)

		for _, vcpu := range vmset.VCpu {
			for _, memory := range vmset.Memory {
				for _, kernel := range vmset.Kernels {
					ip = getNextVMSubnet(ip)
					m, err := buildDomainMatrix(instance.ctx, vcpu, memory, vmset.Name, rc, instance, kernel, fs, ip)
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

func setupDomainVolume(instance *Instance, baseVolumeID, poolName, resourceName string) (*libvirt.Volume, error) {
	volume, err := libvirt.NewVolume(instance.ctx, resourceName, &libvirt.VolumeArgs{
		BaseVolumeId: pulumi.String(baseVolumeID),
		Pool:         pulumi.String(poolName),
		Format:       pulumi.String("qcow2"),
	}, pulumi.Provider(instance.provider))
	if err != nil {
		return nil, err
	}

	return volume, nil
}

func setupLibvirtDomainMatrices(instances map[string]*Instance, vmsets []vmconfig.VMSet, depends []pulumi.Resource) ([]*DomainMatrix, []pulumi.Resource, error) {
	var waitFor []pulumi.Resource
	var matrices []*DomainMatrix

	matrices, waitForMatrices, err := buildDomainMatrices(instances, vmsets, depends)
	if err != nil {
		return matrices, waitFor, err
	}

	for _, instance := range instances {
		instance.provider, err = libvirt.NewProvider(instance.ctx, instance.instanceNamer.ResourceName("provider"), &libvirt.ProviderArgs{
			Uri: instance.libvirtURI,
		}, pulumi.DependsOn(waitForMatrices))
		if err != nil {
			return matrices, waitFor, err
		}
	}

	// setup volumes and domain sockets
	for _, matrix := range matrices {
		instance := matrix.instance
		baseVolumeID := matrix.fs.baseVolumeMap[matrix.kernel.Tag].volumeKey
		volume, err := setupDomainVolume(instance, baseVolumeID, matrix.fs.poolName, matrix.domainNamer.ResourceName("volume"))
		if err != nil {
			return matrices, waitFor, err
		}
		matrix.RecipeLibvirtDomainArgs.Volume = volume

		createDomainSocketDone, err := buildDomainSocket(instance.remoteRunner,
			matrix.domainID,
			matrix.domainNamer.ResourceName("create-domain-socket"),
			depends,
		)
		if err != nil {
			return matrices, waitFor, err
		}
		waitFor = append(waitFor, createDomainSocketDone)
	}

	for _, m := range matrices {
		m.domainArgs = m.RecipeLibvirtDomainArgs.Resources.GetLibvirtDomainArgs(&m.RecipeLibvirtDomainArgs)
	}

	return matrices, waitFor, nil
}

func setupLibvirtVMWithRecipe(instances map[string]*Instance, vmsets []vmconfig.VMSet, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	var dhcpEntries []interface{}
	var newDomainDepends []pulumi.Resource

	matrices, waitForDomainMatrices, err := setupLibvirtDomainMatrices(instances, vmsets, depends)
	if err != nil {
		return []pulumi.Resource{}, err
	}

	networks := make(map[string]*libvirt.Network)
	for arch, instance := range instances {
		for _, m := range matrices {
			if m.instance.Arch == arch {
				dhcpEntries = append(dhcpEntries, m.dhcpEntry)
			}
		}

		network, err := generateNetworkResource(instance.ctx, instance.provider, instance.instanceNamer, dhcpEntries)
		if err != nil {
			return []pulumi.Resource{}, err
		}
		networks[arch] = network

		waitKernelHeaders, err := setupKernelPackages(instance, depends)
		if err != nil {
			return []pulumi.Resource{}, err
		}
		newDomainDepends = append(waitForDomainMatrices, waitKernelHeaders...)

	}

	// attach network interface to each domain
	for _, matrix := range matrices {
		network, ok := networks[matrix.arch]
		if !ok {
			return []pulumi.Resource{}, fmt.Errorf("unsupported arch: %s", matrix.arch)
		}

		matrix.domainArgs.NetworkInterfaces = libvirt.DomainNetworkInterfaceArray{
			libvirt.DomainNetworkInterfaceArgs{
				NetworkId:    network.ID(),
				WaitForLease: pulumi.Bool(false),
			},
		}
	}

	var libvirtDomains []pulumi.Resource
	for _, matrix := range matrices {
		d, err := libvirt.NewDomain(matrix.instance.ctx,
			matrix.domainNamer.ResourceName("ddvm"),
			matrix.domainArgs,
			pulumi.Provider(matrix.instance.provider),
			pulumi.ReplaceOnChanges([]string{"*"}),
			pulumi.DeleteBeforeReplace(true),
			pulumi.DependsOn(newDomainDepends),
		)
		if err != nil {
			return []pulumi.Resource{}, err
		}

		libvirtDomains = append(libvirtDomains, d)
	}

	return libvirtDomains, nil
}
