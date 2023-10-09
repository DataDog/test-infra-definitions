package microvms

import (
	"fmt"
	"net"
	"path/filepath"

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
}

func generateDomainIdentifier(vcpu, memory int, vmsetName, tag, arch string) string {
	return fmt.Sprintf("ddvm-%s-%s-%s-%d-%d", vmsetName, arch, tag, vcpu, memory)
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

type domainConfiguration struct {
	vcpu    int
	memory  int
	setName string
	machine string
	arch    string
	recipe  string
	kernel  vmconfig.Kernel
}

func newDomainConfiguration(e *config.CommonEnvironment, cfg domainConfiguration) (*Domain, error) {
	var err error

	domain := new(Domain)
	domain.domainID = generateDomainIdentifier(cfg.vcpu, cfg.memory, cfg.setName, cfg.kernel.Tag, cfg.arch)
	domain.domainNamer = libvirtResourceNamer(e.Ctx, domain.domainID)

	domain.mac, err = generateMACAddress(e, domain.domainID)
	if err != nil {
		return nil, err
	}

	rc := resources.NewResourceCollection(cfg.recipe)
	domain.RecipeLibvirtDomainArgs.Resources = rc
	domain.RecipeLibvirtDomainArgs.Vcpu = cfg.vcpu
	domain.RecipeLibvirtDomainArgs.Memory = cfg.memory
	domain.RecipeLibvirtDomainArgs.KernelPath = filepath.Join(GetWorkingDirectory(), "kernel-packages", cfg.kernel.Dir, "bzImage")

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
			resources.VCPU:          pulumi.Sprintf("%d", cfg.vcpu),
		},
	)
	domain.RecipeLibvirtDomainArgs.Machine = cfg.machine
	domain.RecipeLibvirtDomainArgs.ExtraKernelParams = cfg.kernel.ExtraParams
	domain.RecipeLibvirtDomainArgs.DomainName = domainName

	return domain, nil
}

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

func GenerateDomainConfigurationsForVMSet(e *config.CommonEnvironment, providerFn LibvirtProviderFn, depends []pulumi.Resource, set *vmconfig.VMSet, fs *LibvirtFilesystem) ([]*Domain, error) {
	var domains []*Domain

	for _, vcpu := range set.VCpu {
		for _, memory := range set.Memory {
			for _, kernel := range set.Kernels {
				domain, err := newDomainConfiguration(
					e,
					domainConfiguration{
						vcpu:    vcpu,
						memory:  memory,
						setName: set.Name,
						machine: set.Machine,
						arch:    set.Arch,
						recipe:  set.Recipe,
						kernel:  kernel,
					},
				)
				if err != nil {
					return []*Domain{}, err
				}

				// setup volume to be used by this domain
				rootVolume, err := setupDomainVolume(
					e.Ctx,
					providerFn,
					depends,
					fs.baseVolumeMap[kernel.Tag].Key(),
					fs.pool.Name(),
					domain.domainNamer.ResourceName("volume"),
				)
				if err != nil {
					return []*Domain{}, err
				}
				domain.RecipeLibvirtDomainArgs.Volumes = append(domain.RecipeLibvirtDomainArgs.Volumes, rootVolume)

				domain.domainArgs = domain.RecipeLibvirtDomainArgs.Resources.GetLibvirtDomainArgs(
					&domain.RecipeLibvirtDomainArgs,
				)

				domains = append(domains, domain)
			}
		}
	}

	return domains, nil

}
