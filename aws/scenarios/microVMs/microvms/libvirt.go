package microvms

import (
	"fmt"
	"net"
	"strings"

	"github.com/DataDog/test-infra-definitions/aws/scenarios/microVMs/microvms/resources"
	"github.com/DataDog/test-infra-definitions/aws/scenarios/microVMs/vmconfig"
	"github.com/DataDog/test-infra-definitions/command"
	"github.com/DataDog/test-infra-definitions/common/namer"
	"github.com/pulumi/pulumi-libvirt/sdk/go/libvirt"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	// The microvm subnet changed from /16 to /24 because the underlying libvirt sdk would identify
	// the incorrect network interface. It looks like it does not respect the subnet range when the subnet
	// used is /16
	// TODO: this problem only manifests when setting up VMs locally. Investigate the root cause to see what can
	// be done. This solution may no longer work when the number of VMs exceeds the ips available in this subnet.
	microVMGroupSubnet    = "169.254.0.0/24"
	domainSocketCreateCmd = `rm -f /tmp/%s.sock && python3 -c "import socket as s; sock = s.socket(s.AF_UNIX); sock.bind('/tmp/%s.sock')"`
)

func generateNetworkResource(ctx *pulumi.Context, provider *libvirt.Provider, depends []pulumi.Resource, resourceNamer namer.Namer, dhcpEntries []interface{}) (*libvirt.Network, error) {

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
	}, pulumi.Provider(provider), pulumi.DeleteBeforeReplace(true), pulumi.DependsOn(depends))
	if err != nil {
		return nil, err
	}

	return network, nil
}

func newLibvirtFS(ctx *pulumi.Context, vmset *vmconfig.VMSet) (*LibvirtFilesystem, error) {
	switch vmset.Recipe {
	case "custom-local":
		fallthrough
	case "custom-arm64":
		fallthrough
	case "custom-amd64":
		return NewLibvirtFSCustomRecipe(ctx, vmset)
	case "distro-local":
		fallthrough
	case "distro-arm64":
		fallthrough
	case "distro-amd64":
		return NewLibvirtFSDistroRecipe(ctx, vmset)
	default:
		return nil, fmt.Errorf("unknown recipe: %s", vmset.Recipe)
	}
}

func buildDomainSocket(runner *Runner, domainID, resourceName string, depends []pulumi.Resource) (pulumi.Resource, error) {
	createDomainSocketArgs := command.Args{
		Create: pulumi.Sprintf(domainSocketCreateCmd, domainID, domainID),
	}
	createDomainSocketDone, err := runner.Command(resourceName, &createDomainSocketArgs, pulumi.DependsOn(depends))
	if err != nil {
		return nil, err
	}

	return createDomainSocketDone, nil
}

func addVMSets(vmsets []vmconfig.VMSet, collection *VMCollection) {
	for _, set := range vmsets {
		if set.Arch == collection.instance.Arch {
			collection.vmsets = append(collection.vmsets, set)
		}
	}
}

// Each VMCollection represents the resources needed to setup a collection of libvirt VMs per instance.
// Each VMCollection corresponds to a single instance
// It is composed of
// instance: The instance on which the components of this VMCollection will be created.
// vmsets: The VMSet which will be part of this collection
// fs: This is the filesystem consisting of pools and basevolumes. Each VMSet will result in 1 filesystems.
// domains: This represents a single libvirt VM. Each VMSet will result in 1 or more domains to be built.
type VMCollection struct {
	instance        *Instance
	vmsets          []vmconfig.VMSet
	fs              map[vmconfig.VMSetID]*LibvirtFilesystem
	domains         []*Domain
	libvirtProvider *libvirt.Provider
}

func (vm *VMCollection) SetupCollectionFilesystems(depends []pulumi.Resource) ([]pulumi.Resource, error) {
	var waitFor []pulumi.Resource

	for _, set := range vm.vmsets {
		fs, err := newLibvirtFS(vm.instance.e.Ctx, &set)
		if err != nil {
			return []pulumi.Resource{}, err
		}

		fsDone, err := fs.SetupLibvirtFilesystem(vm.libvirtProvider, vm.instance.runner, set.Arch, depends)
		if err != nil {
			return []pulumi.Resource{}, err
		}
		waitFor = append(waitFor, fsDone...)

		vm.fs[set.ID] = fs
	}

	return waitFor, nil
}

func (vm *VMCollection) SetupCollectionDomainConfigurations(depends []pulumi.Resource, ip *net.IP) error {
	for _, set := range vm.vmsets {
		fs, ok := vm.fs[set.ID]
		if !ok {
			return fmt.Errorf("failed to find filesystem for vmset %s", set.ID)
		}
		domains, err := GenerateDomainConfigurationsForVMSet(vm.instance.e.CommonEnvironment, vm.libvirtProvider, depends, &set, fs, ip)
		if err != nil {
			return err
		}

		vm.domains = append(vm.domains, domains...)
	}

	return nil
}

func (vm *VMCollection) SetupCollectionNetwork(depends []pulumi.Resource) error {
	var dhcpEntries []interface{}
	var err error

	for _, d := range vm.domains {
		dhcpEntries = append(dhcpEntries, d.dhcpEntry)

	}

	network, err := generateNetworkResource(vm.instance.e.Ctx, vm.libvirtProvider, depends, vm.instance.instanceNamer, dhcpEntries)
	if err != nil {
		return err
	}

	for _, domain := range vm.domains {
		domain.domainArgs.NetworkInterfaces = libvirt.DomainNetworkInterfaceArray{
			libvirt.DomainNetworkInterfaceArgs{
				NetworkId:    network.ID(),
				WaitForLease: pulumi.Bool(false),
			},
		}
	}

	return nil
}

func BuildVMCollections(instances map[string]*Instance, vmsets []vmconfig.VMSet, depends []pulumi.Resource) ([]*VMCollection, []pulumi.Resource, error) {
	var waitFor []pulumi.Resource
	var vmCollections []*VMCollection

	// Map instances and vmsets to VMCollections
	for _, instance := range instances {
		provider, err := libvirt.NewProvider(instance.e.Ctx, instance.instanceNamer.ResourceName("provider"), &libvirt.ProviderArgs{
			Uri: instance.libvirtURI,
		}, pulumi.DependsOn(depends))
		if err != nil {
			return vmCollections, waitFor, err
		}

		collection := &VMCollection{
			fs:              make(map[vmconfig.VMSetID]*LibvirtFilesystem),
			instance:        instance,
			libvirtProvider: provider,
		}

		// add the VMSets required to build this collection
		// This function decides how to partition the sets across collections.
		addVMSets(vmsets, collection)

		vmCollections = append(vmCollections, collection)
	}

	// Setup filesystems, domain configurations, and network
	// for each collection.
	ip, _, _ := net.ParseCIDR(microVMGroupSubnet)
	for _, collection := range vmCollections {
		// setup libvirt filesystem for each collection
		wait, err := collection.SetupCollectionFilesystems(depends)
		if err != nil {
			return vmCollections, waitFor, err
		}
		waitFor = append(waitFor, wait...)

		// build the configurations for all the domains of this collection
		if err := collection.SetupCollectionDomainConfigurations(waitFor, &ip); err != nil {
			return vmCollections, waitFor, err
		}

		// setup domain sockets for communicating with the domains
		for _, domain := range collection.domains {
			createDomainSocketDone, err := buildDomainSocket(collection.instance.runner,
				domain.domainID,
				domain.domainNamer.ResourceName("create-domain-socket", domain.domainID),
				depends,
			)
			if err != nil {
				return vmCollections, waitFor, err
			}
			waitFor = append(waitFor, createDomainSocketDone)
		}

		// setup the network for each collection
		if err := collection.SetupCollectionNetwork(waitFor); err != nil {
			return vmCollections, waitFor, err
		}
	}

	return vmCollections, waitFor, nil

}

func LaunchVMCollections(vmCollections []*VMCollection, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	var libvirtDomains []pulumi.Resource

	for _, collection := range vmCollections {
		for _, domain := range collection.domains {
			d, err := libvirt.NewDomain(collection.instance.e.Ctx,
				domain.domainNamer.ResourceName("ddvm", domain.domainID),
				domain.domainArgs,
				pulumi.Provider(collection.libvirtProvider),
				pulumi.ReplaceOnChanges([]string{"*"}),
				pulumi.DeleteBeforeReplace(true),
				pulumi.DependsOn(depends),
			)
			if err != nil {
				return libvirtDomains, err
			}

			libvirtDomains = append(libvirtDomains, d)
		}
	}

	return libvirtDomains, nil
}

func GetDomainIPMap(vmCollections []*VMCollection) map[string]string {
	ipInformation := make(map[string]string)
	for _, collection := range vmCollections {
		for _, domain := range collection.domains {
			ipInformation[domain.domainID] = domain.ip
		}
	}

	return ipInformation
}
