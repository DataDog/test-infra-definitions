package main

import (
	"fmt"
	"os"

	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/aws/ec2/ec2"
	"github.com/DataDog/test-infra-definitions/command"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi-libvirt/sdk/go/libvirt"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var xlst = `
<xsl:stylesheet version="1.0"
 xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

 <xsl:output omit-xml-declaration="yes"/>

    <xsl:template match="node()|@*">
      <xsl:copy>
         <xsl:apply-templates select="node()|@*"/>
      </xsl:copy>
    </xsl:template>

    <xsl:template match="/domain/devices/controller[@type='usb']">
            <xsl:attribute name="model">
                    <xsl:value-of select="'none'"/>
            </xsl:attribute>
    </xsl:template>


    <xsl:template match="domain/devices/graphics"/>
    <xsl:template match="domain/devices/audio"/>
    <xsl:template match="domain/devices/video"/>
</xsl:stylesheet>
`

func setupLibvirtVM(ctx *pulumi.Context, libvirtUri pulumi.StringOutput, waitForList []pulumi.Resource) error {
	// create a provider, this isn't required, but will make it easier to configure
	// a libvirt_uri, which we'll discuss in a bit
	provider, err := libvirt.NewProvider(ctx, "provider", &libvirt.ProviderArgs{
		Uri: libvirtUri,
	}, pulumi.DependsOn(waitForList))
	if err != nil {
		return err
	}

	// `pool` is a storage pool that can be used to create volumes
	// the `dir` type uses a directory to manage files
	// `Path` maps to a directory on the host filesystem, so we'll be able to
	// volume contents in `/pool/cluster_storage/`
	pool, err := libvirt.NewPool(ctx, "cluster", &libvirt.PoolArgs{
		Type: pulumi.String("dir"),
		Path: pulumi.String("/pool/cluster_storage"),
	}, pulumi.Provider(provider))
	if err != nil {
		return err
	}

	// create a filesystem volume for our VM
	// This filesystem will be based on the `ubuntu` volume above
	// we'll use a size of 10GB
	filesystem, err := libvirt.NewVolume(ctx, "filesystem", &libvirt.VolumeArgs{
		Pool:   pool.Name,
		Source: pulumi.String("/tmp/pulumi-data/bullseye.qcow2"),
		Format: pulumi.String("qcow2"),
	}, pulumi.Provider(provider))
	if err != nil {
		return err
	}

	//	cloud_init_user_data, err := os.ReadFile("./cloud_init_user_data.yaml")
	//	if err != nil {
	//		return err
	//	}
	//
	//	cloud_init_network_config, err := os.ReadFile("./cloud_init_network_config.yaml")
	//	if err != nil {
	//		return err
	//	}
	//
	//	// create a cloud init disk that will setup the ubuntu credentials and enable dhcp
	//	cloud_init, err := libvirt.NewCloudInitDisk(ctx, "cloud-init", &libvirt.CloudInitDiskArgs{
	//		MetaData:      pulumi.String(string(cloud_init_user_data)),
	//		NetworkConfig: pulumi.String(string(cloud_init_network_config)),
	//		Pool:          pool.Name,
	//		UserData:      pulumi.String(string(cloud_init_user_data)),
	//	}, pulumi.Provider(provider))
	//	if err != nil {
	//		return err
	//	}

	// create NAT network using 192.168.10/24 CIDR
	network, err := libvirt.NewNetwork(ctx, "network", &libvirt.NetworkArgs{
		Addresses: pulumi.StringArray{pulumi.String("169.254.0.2/24")},
		Mode:      pulumi.String("nat"),
	}, pulumi.Provider(provider))
	if err != nil {
		return err
	}

	// create a VM that has a name starting with ubuntu
	_, err = libvirt.NewDomain(ctx, "ubuntu", &libvirt.DomainArgs{
		//	Cloudinit: cloud_init.ID(),
		Consoles: libvirt.DomainConsoleArray{
			// enables using `virsh console ...`
			libvirt.DomainConsoleArgs{
				Type:       pulumi.String("pty"),
				TargetPort: pulumi.String("0"),
				TargetType: pulumi.String("serial"),
			},
		},
		Disks: libvirt.DomainDiskArray{
			libvirt.DomainDiskArgs{
				VolumeId: filesystem.ID(),
			},
		},
		NetworkInterfaces: libvirt.DomainNetworkInterfaceArray{
			libvirt.DomainNetworkInterfaceArgs{
				NetworkId:    network.ID(),
				WaitForLease: pulumi.Bool(false),
			},
		},
		Kernel: pulumi.String("/tmp/pulumi-data/bzImage"),
		Cmdlines: pulumi.MapArray{
			pulumi.Map{"console": pulumi.String("ttyS0")},
			pulumi.Map{"acpi": pulumi.String("off")},
			pulumi.Map{"panic": pulumi.String("-1")},
			pulumi.Map{"root": pulumi.String("/dev/sda")},
			pulumi.Map{"net.ifnames": pulumi.String("0")},
			pulumi.Map{"_": pulumi.String("rw")},
		},
		Xml: libvirt.DomainXmlArgs{
			Xslt: pulumi.String(xlst),
		},
		// delete existing VM before creating replacement to avoid two VMs trying to use the same volume
	}, pulumi.Provider(provider), pulumi.ReplaceOnChanges([]string{"*"}), pulumi.DeleteBeforeReplace(true))
	if err != nil {
		return err
	}

	return nil
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		e, err := aws.AWSEnvironment(ctx)
		if err != nil {
			return err
		}

		// boot aws metal instance
		instance, conn, err := ec2.NewDefaultEC2Instance(e, ctx.Stack(), e.DefaultInstanceType())
		if err != nil {
			return err
		}

		// install qemu-kvm and libvirt-daemon-system
		runner, err := command.NewRunner(ctx.Stack()+"-conn", conn, func(r *command.Runner) (*remote.Command, error) {
			return command.WaitForCloudInit(ctx, r)
		})
		if err != nil {
			return err
		}

		aptManager := command.NewAptManager(e.Ctx, runner)
		installQemu, err := aptManager.Ensure("qemu-kvm")
		if err != nil {
			return err
		}
		installLibVirt, err := aptManager.Ensure("libvirt-daemon-system", pulumi.DependsOn([]pulumi.Resource{installQemu}))
		if err != nil {
			return err
		}

		libvirtGroup, err := runner.Command(e.Ctx, "libvirt-group", pulumi.String("sudo adduser $USER libvirt"), nil, nil, nil, true, pulumi.DependsOn([]pulumi.Resource{installLibVirt}))
		if err != nil {
			return err
		}

		disableSELinux, err := runner.Command(e.Ctx, "disable-selinux-qemu", pulumi.String("sudo sed --in-place 's/#security_driver = \"selinux\"/security_driver = \"none\"/' /etc/libvirt/qemu.conf"), nil, nil, nil, true, pulumi.DependsOn([]pulumi.Resource{libvirtGroup}))
		if err != nil {
			return err
		}

		restart, err := runner.Command(e.Ctx, "restart-libvirtd", pulumi.String("sudo systemctl restart libvirtd"), nil, nil, nil, true, pulumi.DependsOn([]pulumi.Resource{disableSELinux}))
		if err != nil {
			return err
		}

		pubKey, err := os.ReadFile("/tmp/test_rsa.pub")
		if err != nil {
			return err
		}

		sshWrite, err := runner.Command(e.Ctx, "write-ssh-key", pulumi.String(fmt.Sprintf("sudo echo \"%s\" >> /home/ubuntu/.ssh/authorized_keys", string(pubKey))), nil, nil, nil, false, pulumi.DependsOn([]pulumi.Resource{instance}))

		url := pulumi.Sprintf("qemu+ssh://ubuntu@%s/system?sshauth=privkey&keyfile=/tmp/test_rsa&known_hosts=/home/ubuntu/.ssh/known_hosts", instance.PrivateIp)
		if err := setupLibvirtVM(e.Ctx, url, []pulumi.Resource{sshWrite, restart}); err != nil {
			return err
		}

		e.Ctx.Export("instance-ip", instance.PrivateIp)
		e.Ctx.Export("connection", conn)

		return nil
	})
}
