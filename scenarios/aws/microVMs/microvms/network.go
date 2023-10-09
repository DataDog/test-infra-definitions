package microvms

import (
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/pulumi/pulumi-libvirt/sdk/go/libvirt"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/DataDog/test-infra-definitions/common/namer"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/microVMs/microvms/resources"
)

// The microvm subnet changed from /16 to /24 because the underlying libvirt sdk would identify
// the incorrect network interface. It looks like it does not respect the subnet range when the subnet
// used is /16.
// TODO: this problem only manifests when setting up VMs locally. Investigate the root cause to see what can
// be done. This solution may no longer work when the number of VMs exceeds the ips available in this subnet.
const microVMGroupSubnetTemplate = "100.%d.0.0/24"

const tcpRPCInfoPorts = "rpcinfo -p | grep -e portmapper -e mountd -e nfs | grep tcp | rev | cut -d ' ' -f 3 | rev | sort | uniq | tr '\n' ' ' | awk '{$1=$1};1' | tr ' ' ',' | tr -d '\n'"
const udpRPCInfoPorts = "rpcinfo -p | grep -e portmapper -e mountd -e nfs | grep udp | rev | cut -d ' ' -f 3 | rev | sort | uniq | tr '\n' ' ' | awk '{$1=$1};1' | tr ' ' ',' | tr -d '\n'"

const iptablesDeleteRuleFlag = "-D"
const iptablesAddRuleFlag = "-A"

const iptablesTCPRule = "iptables %s INPUT -p tcp -i %s -s %s -m multiport --dports $(%s) -m state --state NEW,ESTABLISHED -j ACCEPT"
const iptablesUDPRule = "iptables %s INPUT -p udp -i %s -s %s -m multiport --dports $(%s) -j ACCEPT"

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

func getMicroVMGroupSubnetPattern(subnet string) string {
	ip, _, _ := net.ParseCIDR(subnet)
	ipv4 := ip.To4()
	// this assumes a /24
	return fmt.Sprintf("%d.%d.%d.*", ipv4[0], ipv4[1], ipv4[2])
}

func getMicroVMGroupSubnet() (string, error) {
	for i := 1; i < 254; i++ {
		subnet := fmt.Sprintf(microVMGroupSubnetTemplate, i)
		if free, err := freeSubnet(subnet); err == nil && free {
			return subnet, nil
		}
	}

	return "", fmt.Errorf("getMicroVMGroupSubnet: could not find subnet")
}

func allowNFSPortsForBridge(ctx *pulumi.Context, isLocal bool, bridge pulumi.StringOutput, runner *Runner, resourceNamer namer.Namer) ([]pulumi.Resource, error) {
	sudoPassword := GetSudoPassword(ctx, isLocal)
	iptablesAllowTCPArgs := command.Args{
		Create:                   pulumi.Sprintf(iptablesTCPRule, iptablesAddRuleFlag, bridge, microVMGroupSubnet, tcpRPCInfoPorts),
		Delete:                   pulumi.Sprintf(iptablesTCPRule, iptablesDeleteRuleFlag, bridge, microVMGroupSubnet, tcpRPCInfoPorts),
		Sudo:                     true,
		RequirePasswordFromStdin: true,
		Stdin:                    sudoPassword,
	}
	iptablesAllowTCPDone, err := runner.Command(resourceNamer.ResourceName("allow-nfs-ports-tcp"), &iptablesAllowTCPArgs)
	if err != nil {
		return []pulumi.Resource{}, err
	}

	iptablesAllowUDPArgs := command.Args{
		Create:                   pulumi.Sprintf(iptablesUDPRule, iptablesAddRuleFlag, bridge, microVMGroupSubnet, udpRPCInfoPorts),
		Delete:                   pulumi.Sprintf(iptablesUDPRule, iptablesDeleteRuleFlag, bridge, microVMGroupSubnet, udpRPCInfoPorts),
		Sudo:                     true,
		RequirePasswordFromStdin: true,
		Stdin:                    sudoPassword,
	}
	iptablesAllowUDPDone, err := runner.Command(resourceNamer.ResourceName("allow-nfs-ports-udp"), &iptablesAllowUDPArgs)
	if err != nil {
		return []pulumi.Resource{}, err
	}

	return []pulumi.Resource{iptablesAllowTCPDone, iptablesAllowUDPDone}, nil
}

func generateNetworkResource(ctx *pulumi.Context, providerFn LibvirtProviderFn, depends []pulumi.Resource, resourceNamer namer.Namer, dhcpEntries []interface{}) (*libvirt.Network, error) {
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

	provider, err := providerFn()
	if err != nil {
		return nil, err
	}

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
