package microvms

import (
	"fmt"
	"net"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/pulumi/pulumi-libvirt/sdk/go/libvirt"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/DataDog/test-infra-definitions/common/namer"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/microVMs/microvms/resources"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/microVMs/vmconfig"
)

const domainSocketCreateCmd = `rm -f /tmp/%s.sock && python3 -c "import socket as s; sock = s.socket(s.AF_UNIX); sock.bind('/tmp/%s.sock')"`

func libvirtResourceName(identifiers ...string) string {
	return strings.Join(identifiers, "-")
}

func libvirtResourceNamer(ctx *pulumi.Context, identifiers ...string) namer.Namer {
	return namer.NewNamer(ctx, libvirtResourceName(identifiers...))
}

type LibvirtProviderFn func() (*libvirt.Provider, error)

func newLibvirtFS(ctx *pulumi.Context, vmset *vmconfig.VMSet, pools map[vmconfig.PoolType]LibvirtPool) (*LibvirtFilesystem, error) {
	switch vmset.Recipe {

	case vmconfig.RecipeCustomLocal:
		fallthrough
	case vmconfig.RecipeCustomARM64:
		fallthrough
	case vmconfig.RecipeCustomAMD64:
		return NewLibvirtFSCustomRecipe(ctx, vmset, pools), nil
	case vmconfig.RecipeDistroLocal:
		fallthrough
	case vmconfig.RecipeDistroARM64:
		fallthrough
	case vmconfig.RecipeDistroAMD64:
		return NewLibvirtFSDistroRecipe(ctx, vmset, pools), nil
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

type LibvirtProvider struct {
	libvirtProviderFn   LibvirtProviderFn
	initLibvirtProvider sync.Once
	provider            *libvirt.Provider
}

// Each VMCollection represents the resources needed to setup a collection of libvirt VMs per instance.
// Each VMCollection corresponds to a single instance
// It is composed of
// instance: The instance on which the components of this VMCollection will be created.
// vmsets: The VMSet which will be part of this collection
// pool: The libvirt pool which will be shared across all vmsets in the collection.
// fs: This is the filesystem consisting of pools and basevolumes. Each VMSet will result in 1 filesystems.
// domains: This represents a single libvirt VM. Each VMSet will result in 1 or more domains to be built.
type VMCollection struct {
	instance *Instance
	vmsets   []vmconfig.VMSet
	pools    map[vmconfig.PoolType]LibvirtPool
	fs       map[vmconfig.VMSetID]*LibvirtFilesystem
	domains  []*Domain
	LibvirtProvider
}

func (vm *VMCollection) SetupCollectionFilesystems(depends []pulumi.Resource) ([]pulumi.Resource, error) {
	var waitFor []pulumi.Resource

	for _, pool := range vm.pools {
		libvirtPoolReady, err := pool.SetupLibvirtPool(
			vm.instance.e.Ctx,
			vm.instance.runner,
			vm.libvirtProviderFn,
			vm.instance.Arch == LocalVMSet,
			depends,
		)
		if err != nil {
			return nil, err
		}
		depends = append(depends, libvirtPoolReady...)
	}

	for _, set := range vm.vmsets {
		fs, err := newLibvirtFS(vm.instance.e.Ctx, &set, vm.pools)
		if err != nil {
			return nil, err
		}
		vm.fs[set.ID] = fs
	}

	// Duplicate VMs maybe be booted in different VMSets.
	// In order to avoid downloading and building the baseVolumes twice,
	// we prune the list of `volumes`.
	seen := make(map[string]bool)
	for _, fs := range vm.fs {
		imagesToKeep := []LibvirtVolume{}
		for _, volume := range fs.volumes {
			fsImage := volume.UnderlyingImage()
			if present, _ := seen[fsImage.imagePath]; present {
				continue
			}
			imagesToKeep = append(imagesToKeep, volume)

			seen[fsImage.imagePath] = true
		}
		fs.volumes = imagesToKeep
	}

	for _, fs := range vm.fs {
		fsDone, err := fs.SetupLibvirtFilesystem(vm.libvirtProviderFn, vm.instance.runner, depends)
		if err != nil {
			return nil, err
		}
		waitFor = append(waitFor, fsDone...)
	}

	return waitFor, nil
}

func (vm *VMCollection) SetupCollectionDomainConfigurations(depends []pulumi.Resource) ([]pulumi.Resource, error) {
	var waitFor []pulumi.Resource

	for _, set := range vm.vmsets {
		fs, ok := vm.fs[set.ID]
		if !ok {
			return nil, fmt.Errorf("failed to find filesystem for vmset %s", set.ID)
		}
		domains, err := GenerateDomainConfigurationsForVMSet(vm.instance.e.CommonEnvironment, vm.libvirtProviderFn, depends, &set, fs)
		if err != nil {
			return nil, err
		}

		// Setup individual Nvram disk for arm64 distro images
		if resources.GetLocalArchRecipe(set.Recipe) == vmconfig.RecipeDistroARM64 {
			for _, domain := range domains {
				varstorePath := filepath.Join(GetWorkingDirectory(), fmt.Sprintf("varstore.%s", domain.DomainName))
				varstoreArgs := command.Args{
					Create: pulumi.Sprintf("truncate -s 64m %s", varstorePath),
					Delete: pulumi.Sprintf("rm -f %s", varstorePath),
				}
				varstoreDone, err := vm.instance.runner.Command(
					domain.domainNamer.ResourceName("create-nvram"),
					&varstoreArgs,
					pulumi.DependsOn(depends),
				)
				if err != nil {
					return nil, err
				}

				waitFor = append(waitFor, varstoreDone)
			}
		}

		vm.domains = append(vm.domains, domains...)
	}

	return waitFor, nil
}

func (vm *VMCollection) SetupCollectionNetwork(depends []pulumi.Resource) error {
	var dhcpEntries []interface{}
	var err error

	for _, d := range vm.domains {
		dhcpEntries = append(dhcpEntries, d.dhcpEntry)

	}

	network, err := generateNetworkResource(vm.instance.e.Ctx, vm.libvirtProviderFn, depends, vm.instance.instanceNamer, dhcpEntries)
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

	// set iptable rules for allowing ports to access NFS server
	_, err = allowNFSPortsForBridge(vm.instance.e.Ctx, vm.instance.Arch == LocalVMSet, network.Bridge, vm.instance.runner, vm.instance.instanceNamer)
	if err != nil {
		return err
	}

	return nil
}

func buildLibvirtProviderFn(collection *VMCollection, depends []pulumi.Resource) func() (*libvirt.Provider, error) {
	var err error
	return func() (*libvirt.Provider, error) {
		collection.initLibvirtProvider.Do(func() {
			collection.provider, err = libvirt.NewProvider(
				collection.instance.e.Ctx,
				collection.instance.instanceNamer.ResourceName("libvirt-provider"),
				&libvirt.ProviderArgs{
					Uri: collection.instance.libvirtURI,
				}, pulumi.DependsOn(depends))
		})
		if err != nil {
			return nil, err
		}

		return collection.provider, nil
	}
}

func buildCollectionPools(ctx *pulumi.Context, collection *VMCollection) error {
	if len(collection.vmsets) == 0 {
		return ErrVMSetsNotMapped
	}

	collection.pools = make(map[vmconfig.PoolType]LibvirtPool)
	collection.pools[resources.DefaultPool] = NewGlobalLibvirtPool(ctx)

	var err error
	for _, v := range collection.vmsets {
		for _, d := range v.Disks {
			switch d.Type {
			case resources.RAMPool:
				if _, ok := collection.pools[resources.RAMPool]; !ok {
					collection.pools[resources.RAMPool], err = NewRAMBackedLibvirtPool(ctx, &d)
				}
			default:
			}

			if err != nil {
				return err
			}
		}
	}

	return nil
}

func BuildVMCollections(instances map[string]*Instance, vmsets []vmconfig.VMSet, depends []pulumi.Resource) ([]*VMCollection, []pulumi.Resource, error) {
	var waitFor []pulumi.Resource
	var vmCollections []*VMCollection

	// Map instances and vmsets to VMCollections
	for _, instance := range instances {
		collection := &VMCollection{
			fs:       make(map[vmconfig.VMSetID]*LibvirtFilesystem),
			instance: instance,
		}

		// We want to lazily initialize the libvirt provider.
		// If there is too long a time between the provider being initialized,
		// which essentially creates a connection with the libvirt daemon, and the provider
		// created connection being used, the libvirt daemon drops the connection
		// causing our scenario to fail.
		collection.libvirtProviderFn = buildLibvirtProviderFn(collection, depends)

		// add the VMSets required to build this collection
		// This function decides how to partition the sets across collections.
		addVMSets(vmsets, collection)

		// builds pools once sets have been mapped
		if err := buildCollectionPools(instance.e.Ctx, collection); err != nil {
			return vmCollections, waitFor, err
		}

		vmCollections = append(vmCollections, collection)
	}

	// Setup filesystems, domain configurations, and network
	// for each collection.
	for _, collection := range vmCollections {
		// setup libvirt filesystem for each collection
		wait, err := collection.SetupCollectionFilesystems(depends)
		if err != nil {
			return vmCollections, waitFor, err
		}
		waitFor = append(waitFor, wait...)

		// build the configurations for all the domains of this collection
		wait, err = collection.SetupCollectionDomainConfigurations(waitFor)
		if err != nil {
			return vmCollections, waitFor, err
		}
		waitFor = append(waitFor, wait...)
	}

	// map domains to ips
	var domains []*Domain
	for _, collection := range vmCollections {
		domains = append(domains, collection.domains...)
	}
	// Sort the domains so the ips are mapped deterministically across updates
	// Otherwise an update could compute the mapping to be different even if nothing
	// changed. This will result in updated DHCP entries in the network. This breaks
	// the CI since the mapping is saved once on setup and not refreshed after pulumi update.
	// If the ips drift across instances the CI job will end up attempting to connect
	// to VMs that do no exist on the target instance.
	sort.Slice(domains, func(i, j int) bool {
		return domains[i].domainID < domains[j].domainID
	})

	// Discover subnet to use for the network.
	// This is done dynamically so we can have concurrent micro-vm groups
	// active, without the network conflicting.
	var err error
	initMicroVMGroupSubnet.Do(func() {
		microVMGroupSubnet, err = getMicroVMGroupSubnet()
	})
	if err != nil {
		return vmCollections, waitFor, fmt.Errorf("generateNetworkResource: unable to find any free subnet")
	}
	ip, _, _ := net.ParseCIDR(microVMGroupSubnet)
	// The first ip address is derived from the microvm subnet.
	// The gateway ip address is xxx.yyy.zzz.1. So the first VM should have address xxx.yyy.zzz.2
	// Therefore we call getNextVMIP consecutively to move start from xxx.yyy.zzz.2
	ip = getNextVMIP(&ip)
	for _, d := range domains {
		ip = getNextVMIP(&ip)
		d.ip = fmt.Sprintf("%s", ip)
		d.dhcpEntry = generateDHCPEntry(d.mac, d.ip, d.domainID)
	}

	// Network setup has to be done after the dhcp entries have been generated for each domain
	for _, collection := range vmCollections {
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
		provider, err := collection.libvirtProviderFn()
		if err != nil {
			return libvirtDomains, err
		}
		for _, domain := range collection.domains {
			d, err := libvirt.NewDomain(collection.instance.e.Ctx,
				domain.domainNamer.ResourceName(domain.domainID),
				domain.domainArgs,
				pulumi.Provider(provider),
				pulumi.ReplaceOnChanges([]string{"*"}),
				pulumi.DeleteBeforeReplace(true),
				pulumi.DependsOn(depends),
				// Pulumi incorrectly detects these as changing everytime.
				pulumi.IgnoreChanges([]string{"filesystems", "firmware", "nvram"}),
			)
			if err != nil {
				return libvirtDomains, err
			}
			domain.lvDomain = d

			libvirtDomains = append(libvirtDomains, d)
		}
	}

	return libvirtDomains, nil
}
