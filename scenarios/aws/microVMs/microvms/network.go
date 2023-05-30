package microvms

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/DataDog/test-infra-definitions/common/namer"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/microVMs/microvms/resources"
	"github.com/pulumi/pulumi-libvirt/sdk/go/libvirt"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// The microvm subnet changed from /16 to /24 because the underlying libvirt sdk would identify
// the incorrect network interface. It looks like it does not respect the subnet range when the subnet
// used is /16.
// TODO: this problem only manifests when setting up VMs locally. Investigate the root cause to see what can
// be done. This solution may no longer work when the number of VMs exceeds the ips available in this subnet.
const microVMGroupSubnetTemplate = "%d.254.0.0/24"

var initMicroVMGroupSubnet sync.Once
var microVMGroupSubnet string

func freeSubnet(subnet string) (bool, error) {
	startIP, _, err := net.ParseCIDR(subnet)
	if err != nil {
		return false, err
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return false, err
	}

	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			switch v := a.(type) {
			case *net.IPNet:
				if v.Contains(startIP) {
					return false, nil
				}
			}
		}
	}

	return true, nil
}

func getMicroVMGroupSubnet() (string, error) {
	for i := 100; i < 254; i++ {
		subnet := fmt.Sprintf(microVMGroupSubnetTemplate, i)
		if free, err := freeSubnet(subnet); err == nil && free {
			return subnet, nil
		}
	}

	return "", errors.New("getMicroVMGroupSubnet: could not find subnet")
}

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
