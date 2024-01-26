package microvms

import (
	"fmt"
	"net"
	"path/filepath"
	"strings"

	"github.com/pulumi/pulumi-libvirt/sdk/go/libvirt"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/namer"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/microVMs/microvms/resources"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/microVMs/vmconfig"
)

const (
	dhcpEntriesTemplate = "<host mac='%s' name='%s' ip='%s'/>"
	sharedFSMountPoint  = "/opt/kernel-version-testing"
)

func getNextVMIP(ip *net.IP) net.IP {
	ipv4 := ip.To4()
	ipv4[3]++

	return ipv4
}

type Domain struct {
	resources.RecipeLibvirtDomainArgs
	domainID    string
	dhcpEntry   pulumi.StringOutput
	domainArgs  *libvirt.DomainArgs
	domainNamer namer.Namer
	ip          string
	mac         pulumi.StringOutput
	lvDomain    *libvirt.Domain
	tag         string
	vmset       vmconfig.VMSet
}

func generateDomainIdentifier(vcpu, memory int, vmsetTags, tag, arch string) string {
	// The domain id should always begin with 'arch'-'tag'-'vmsetTags'. This order
	// is expected in the consumers of this framework
	return fmt.Sprintf("%s-%s-%s-ddvm-%d-%d", arch, tag, vmsetTags, vcpu, memory)
}
func generateNewUnicastMac(e config.CommonEnvironment, domainID string) (pulumi.StringOutput, error) {
	r := utils.NewRandomGenerator(e, domainID)

	pulumiRandStr, err := r.RandomString(domainID, 6, true)
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

func generateMACAddress(e *config.CommonEnvironment, domainID string) (pulumi.StringOutput, error) {
	mac, err := generateNewUnicastMac(*e, domainID)
	if err != nil {
		return mac, err
	}

	return mac, err
}

func generateDHCPEntry(mac pulumi.StringOutput, ip, domainID string) pulumi.StringOutput {
	return pulumi.Sprintf(dhcpEntriesTemplate, mac, domainID, ip)
}

func getCPUTuneXML(vmcpus, hostCPUSet, cpuCount int) (string, int) {
	var vcpuMap []string

	if cpuCount == 0 {
		return "", 0
	}

	for i := 0; i < vmcpus; i++ {
		vcpuMap = append(vcpuMap, fmt.Sprintf("<vcpupin vcpu='%d' cpuset='%d'/>", i, hostCPUSet))
		hostCPUSet++
		if hostCPUSet >= cpuCount {
			hostCPUSet = 0
		}
	}

	return fmt.Sprintf("<cputune>%s</cputune>", strings.Join(vcpuMap, "\n")), hostCPUSet
}

func newDomainConfiguration(e *config.CommonEnvironment, set *vmconfig.VMSet, vcpu, memory int, kernel vmconfig.Kernel, cputune string) (*Domain, error) {
	var err error

	domain := new(Domain)
	setTags := strings.Join(set.Tags, "-")
	domain.domainID = generateDomainIdentifier(vcpu, memory, setTags, kernel.Tag, set.Arch)
	domain.domainNamer = libvirtResourceNamer(e.Ctx, domain.domainID)
	domain.tag = kernel.Tag
	// copy the vmset tag. The pointer refers to
	// a local variable and can change causing an incorrect mapping
	domain.vmset = *set

	domain.mac, err = generateMACAddress(e, domain.domainID)
	if err != nil {
		return nil, err
	}

	rc := resources.NewResourceCollection(set.Recipe)
	domain.RecipeLibvirtDomainArgs.Resources = rc
	domain.RecipeLibvirtDomainArgs.Vcpu = vcpu
	domain.RecipeLibvirtDomainArgs.Memory = memory
	domain.RecipeLibvirtDomainArgs.ConsoleType = set.ConsoleType
	domain.RecipeLibvirtDomainArgs.KernelPath = filepath.Join(GetWorkingDirectory(), "kernel-packages", kernel.Dir, "bzImage")

	domainName := libvirtResourceName(e.Ctx.Stack(), domain.domainID)
	varstore := filepath.Join(GetWorkingDirectory(), fmt.Sprintf("varstore.%s", domainName))
	efi := filepath.Join(GetWorkingDirectory(), "efi.fd")
	domain.RecipeLibvirtDomainArgs.Xls = rc.GetDomainXLS(
		map[string]pulumi.StringInput{
			resources.SharedFSMount: pulumi.String(sharedFSMountPoint),
			resources.DomainID:      pulumi.String(domain.domainID),
			resources.MACAddress:    domain.mac,
			resources.Nvram:         pulumi.String(varstore),
			resources.Efi:           pulumi.String(efi),
			resources.VCPU:          pulumi.Sprintf("%d", vcpu),
			resources.CPUTune:       pulumi.String(cputune),
		},
	)
	domain.RecipeLibvirtDomainArgs.Machine = set.Machine
	domain.RecipeLibvirtDomainArgs.ExtraKernelParams = kernel.ExtraParams
	domain.RecipeLibvirtDomainArgs.DomainName = domainName

	return domain, nil
}

// We create a final overlay here so that each VM has its own unique writable disk.
// At this stage the chain of images is as follows: base-image -> overlay-1
// After this function we have: base-image -> overlay-1 -> overlay-2
// We have to do this because we have as many overlay-1's as the number of unique base images.
// However, we may want multiple VMs booted from the same underlying filesystem. To support this
// case we create a final overlay-2 for each VM to boot from.
func setupDomainVolume(ctx *pulumi.Context, providerFn LibvirtProviderFn, depends []pulumi.Resource, baseVolumeID, poolName, resourceName string) (*libvirt.Volume, error) {
	provider, err := providerFn()
	if err != nil {
		return nil, err
	}

	volume, err := libvirt.NewVolume(ctx, resourceName, &libvirt.VolumeArgs{
		BaseVolumeId: pulumi.String(baseVolumeID),
		Pool:         pulumi.String(poolName),
		Format:       pulumi.String("qcow2"),
	}, pulumi.Provider(provider), pulumi.DependsOn(depends))
	if err != nil {
		return nil, err
	}

	return volume, nil
}

func getVolumeDiskTarget(isRootVolume bool, lastDisk string) string {
	if isRootVolume {
		return "/dev/vda"
	}

	return fmt.Sprintf("/dev/vd%c", rune(int(lastDisk[len(lastDisk)-1])+1))
}

func GenerateDomainConfigurationsForVMSet(e *config.CommonEnvironment, providerFn LibvirtProviderFn, depends []pulumi.Resource, set *vmconfig.VMSet, fs *LibvirtFilesystem, cpuSetStart int) ([]*Domain, int, error) {
	var domains []*Domain
	var cpuTuneXML string

	for _, vcpu := range set.VCpu {
		for _, memory := range set.Memory {
			for _, kernel := range set.Kernels {
				cpuTuneXML, cpuSetStart = getCPUTuneXML(vcpu, cpuSetStart, set.VMHost.AvailableCPUs)
				domain, err := newDomainConfiguration(e, set, vcpu, memory, kernel, cpuTuneXML)
				if err != nil {
					return []*Domain{}, 0, err
				}

				// setup volume to be used by this domain
				libvirtVolumes := fs.baseVolumeMap[kernel.Tag]
				lastDisk := getVolumeDiskTarget(true, "")
				for _, vol := range libvirtVolumes {
					lastDisk = getVolumeDiskTarget(vol.Mountpoint() == RootMountpoint, lastDisk)
					rootVolume, err := setupDomainVolume(
						e.Ctx,
						providerFn,
						depends,
						vol.Key(),
						vol.Pool().Name(),
						vol.FullResourceName("final-overlay", kernel.Tag),
					)
					if err != nil {
						return []*Domain{}, 0, err
					}
					domain.Disks = append(domain.Disks, resources.DomainDisk{
						VolumeID:   pulumi.StringPtrInput(rootVolume.ID()),
						Target:     lastDisk,
						Mountpoint: vol.Mountpoint(),
					})
				}

				domain.domainArgs, err = domain.RecipeLibvirtDomainArgs.Resources.GetLibvirtDomainArgs(
					&domain.RecipeLibvirtDomainArgs,
				)
				if err != nil {
					return []*Domain{}, 0, fmt.Errorf("failed to setup domain arguments for %s: %v", domain.domainID, err)
				}

				domains = append(domains, domain)
			}
		}
	}

	return domains, cpuSetStart, nil

}
