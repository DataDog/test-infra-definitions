package microvms

import (
	"fmt"
	"net"
	"path/filepath"

	"github.com/DataDog/test-infra-definitions/aws/scenarios/microVMs/microvms/resources"
	"github.com/DataDog/test-infra-definitions/aws/scenarios/microVMs/vmconfig"
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/namer"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/pulumi/pulumi-libvirt/sdk/go/libvirt"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const dhcpEntriesTemplate = "<host mac='%s' name='%s' ip='%s'/>"

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

func generateDHCPEntry(e *config.CommonEnvironment, ip, domainID string) (pulumi.StringOutput, pulumi.StringOutput, error) {
	mac, err := generateNewUnicastMac(*e, domainID)
	if err != nil {
		return pulumi.StringOutput{}, mac, err
	}

	return pulumi.Sprintf(dhcpEntriesTemplate, mac, domainID, ip), mac, nil
}

func newDomainConfiguration(e *config.CommonEnvironment, vcpu, memory int, setName, machine, arch, ip string, kernel vmconfig.Kernel, recipe string) (*Domain, error) {
	var mac pulumi.StringOutput
	var err error

	domain := new(Domain)
	domain.domainID = generateDomainIdentifier(vcpu, memory, setName, kernel.Tag, arch)
	domain.domainNamer = libvirtResourceNamer(e.Ctx, domain.domainID)

	domain.ip = ip
	domain.dhcpEntry, mac, err = generateDHCPEntry(e, ip, domain.domainID)
	if err != nil {
		return nil, err
	}

	rc := resources.NewResourceCollection(recipe)
	domain.RecipeLibvirtDomainArgs.Resources = rc
	domain.RecipeLibvirtDomainArgs.Vcpu = vcpu
	domain.RecipeLibvirtDomainArgs.Memory = memory
	domain.RecipeLibvirtDomainArgs.KernelPath = filepath.Join(GetWorkingDirectory(), "kernel-packages", kernel.Dir, "bzImage")
	domain.RecipeLibvirtDomainArgs.Xls = rc.GetDomainXLS(
		map[string]pulumi.StringInput{
			resources.SharedFSMount: pulumi.String(sharedFSMountPoint),
			resources.DomainID:      pulumi.String(domain.domainID),
			resources.MACAddress:    mac,
		},
	)
	domain.RecipeLibvirtDomainArgs.Machine = machine
	domain.RecipeLibvirtDomainArgs.ExtraKernelParams = kernel.ExtraParams
	domain.RecipeLibvirtDomainArgs.DomainName = domain.domainID

	return domain, nil
}

func setupDomainVolume(ctx *pulumi.Context, provider *libvirt.Provider, depends []pulumi.Resource, baseVolumeID, poolName, resourceName string) (*libvirt.Volume, error) {
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

func GenerateDomainConfigurationsForVMSet(e *config.CommonEnvironment, provider *libvirt.Provider, depends []pulumi.Resource, set *vmconfig.VMSet, fs *LibvirtFilesystem, ip *net.IP) ([]*Domain, error) {
	var domains []*Domain

	for _, vcpu := range set.VCpu {
		for _, memory := range set.Memory {
			for _, kernel := range set.Kernels {
				*ip = getNextVMIP(ip)
				domain, err := newDomainConfiguration(
					e, vcpu,
					memory, set.Name,
					set.Machine, set.Arch,
					fmt.Sprintf("%s", ip), kernel,
					set.Recipe,
				)
				if err != nil {
					return []*Domain{}, err
				}

				// setup volume to be used by this domain
				domain.RecipeLibvirtDomainArgs.Volume, err = setupDomainVolume(
					e.Ctx,
					provider,
					depends,
					fs.baseVolumeMap[kernel.Tag].volumeKey,
					fs.poolName,
					domain.domainNamer.ResourceName("volume"),
				)
				if err != nil {
					return []*Domain{}, err
				}

				domain.domainArgs = domain.RecipeLibvirtDomainArgs.Resources.GetLibvirtDomainArgs(
					&domain.RecipeLibvirtDomainArgs,
				)

				domains = append(domains, domain)
			}
		}
	}

	return domains, nil

}
