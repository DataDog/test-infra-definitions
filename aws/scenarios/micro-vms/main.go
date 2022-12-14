package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"

	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/aws/ec2/ec2"
	"github.com/DataDog/test-infra-definitions/aws/scenarios/micro-vms/vmconfig"
	"github.com/DataDog/test-infra-definitions/command"
	awsEc2 "github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi-libvirt/sdk/go/libvirt"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"golang.org/x/crypto/ssh"
)

var libvirtPrivateKey string

func generateSSHKeyPair() (privateKey []byte, publicKey []byte, err error) {
	priv, err := generatePrivateKey()
	if err != nil {
		return
	}

	publicKey, err = generatePublicKey(&priv.PublicKey)
	if err != nil {
		return
	}

	privateKey = encodePrivateKeyToPEM(priv)

	return
}

func encodePrivateKeyToPEM(privateKey *rsa.PrivateKey) []byte {
	privDER := x509.MarshalPKCS1PrivateKey(privateKey)

	privBlock := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   privDER,
	}

	privatePEM := pem.EncodeToMemory(&privBlock)

	return privatePEM
}

func generatePrivateKey() (*rsa.PrivateKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	err = privateKey.Validate()
	if err != nil {
		return nil, err
	}

	return privateKey, nil
}

func generatePublicKey(privatekey *rsa.PublicKey) ([]byte, error) {
	publicRsaKey, err := ssh.NewPublicKey(privatekey)
	if err != nil {
		return nil, err
	}

	pubKeyBytes := ssh.MarshalAuthorizedKey(publicRsaKey)

	return pubKeyBytes, nil
}

func writeKeyToTempFile(keyBytes []byte, targetFile string) (string, error) {
	f, err := os.CreateTemp("", targetFile)
	if err != nil {
		return "", err
	}
	defer f.Close()

	_, err = f.Write(keyBytes)
	if err != nil {
		return "", err
	}

	return f.Name(), nil
}

func newMetalInstance(e aws.Environment, name string) (*awsEc2.Instance, remote.ConnectionOutput, error) {
	fmt.Printf("instance: %v\n", e.DefaultInstanceType())
	awsInstance, conn, err := ec2.NewDefaultEC2Instance(e, name, e.DefaultInstanceType())
	if err != nil {
		return nil, remote.ConnectionOutput{}, err
	}

	return awsInstance, conn, err
}

var poolXml = `
<pool type='dir'>
  <name>cluster_storage</name>
  <capacity unit='bytes'>0</capacity>
  <allocation unit='bytes'>0</allocation>
  <available unit='bytes'>0</available>
  <source>
  </source>
  <target>
    <path>/pool/cluster_storage</path>
	<permissions>
      <owner>$(cat /etc/passwd | grep libvirt-qemu | cut -d ':' -f 3)</owner>
	  <group>$(cat /etc/group | grep kvm | cut -d ':' -f 3)</group>
      <mode>0777</mode>
    </permissions>
  </target>
</pool>
`

var volXml = `
<volume type='file'>
  <name>bullseye-base</name>
  <key>/pool/cluster_storage/bullseye-base</key>
  <capacity unit='bytes'>10737418240</capacity>
  <allocation unit='bytes'>10000007168</allocation>
  <physical unit='bytes'>10000000000</physical>
  <target>
    <path>/pool/cluster_storage/bullseye-base</path>
    <format type='qcow2'/>
    <permissions>
      <mode>0666</mode>
      <owner>64055</owner>
      <group>109</group>
    </permissions>
    <compat>1.1</compat>
    <clusterSize unit='B'>65536</clusterSize>
    <features/>
  </target>
</volume>
`

func provisionInstance(vm *awsEc2.Instance, conn remote.ConnectionOutput, e aws.Environment) ([]pulumi.Resource, error) {
	runner, err := command.NewRunner(*e.CommonEnvironment, e.Ctx.Stack()+"-conn", conn, func(r *command.Runner) (*remote.Command, error) {
		return command.WaitForCloudInit(e.Ctx, r)
	})

	aptManager := command.NewAptManager(runner)
	installQemu, err := aptManager.Ensure("qemu-kvm")
	if err != nil {
		return []pulumi.Resource{}, err
	}

	downloadKernel, err := runner.Command("download-kernel-image", pulumi.String("wget -q https://dd-agent-omnibus.s3.amazonaws.com/bzImage -O /tmp/bzImage"), nil, nil, nil, false)
	downloadRootfs, err := runner.Command("download-rootfs", pulumi.String("wget -q https://dd-agent-omnibus.s3.amazonaws.com/rootfs.tar.gz -O /tmp/rootfs.tar.gz"), nil, nil, nil, false)
	extractRootfs, err := runner.Command("extract-rootfs", pulumi.String("tar xzOf /tmp/rootfs.tar.gz > /tmp/bullseye.qcow2"), nil, nil, nil, false, pulumi.DependsOn([]pulumi.Resource{downloadRootfs}))
	//	convertRootfs, err := runner.Command("convert-rootfs", pulumi.String("qemu-img convert /tmp/bullseye.qcow2 /tmp/bullseye.img"), nil, nil, nil, false, pulumi.DependsOn([]pulumi.Resource{extractRootfs}))

	installLibVirt, err := aptManager.Ensure("libvirt-daemon-system", pulumi.DependsOn([]pulumi.Resource{installQemu}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	disableSELinux, err := runner.Command("disable-selinux-qemu", pulumi.String("sed --in-place 's/#security_driver = \"selinux\"/security_driver = \"none\"/' /etc/libvirt/qemu.conf"), nil, nil, nil, true, pulumi.DependsOn([]pulumi.Resource{installLibVirt}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	libvirtReady, err := runner.Command("restart-libvirtd", pulumi.String("systemctl restart libvirtd"), nil, nil, nil, true, pulumi.DependsOn([]pulumi.Resource{disableSELinux}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	// build volume
	poolXmlWritten, err := runner.Command("write-pool-xml", pulumi.String(fmt.Sprintf("echo \"%s\" > /tmp/pool.xml", poolXml)), nil, nil, nil, false, pulumi.DependsOn([]pulumi.Resource{libvirtReady}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	poolDefineReady, err := runner.Command("define-libvirt-pool", pulumi.String("virsh pool-define /tmp/pool.xml"), nil, nil, nil, true, pulumi.DependsOn([]pulumi.Resource{poolXmlWritten}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	poolBuildReady, err := runner.Command("build-libvirt-pool", pulumi.String("virsh pool-build cluster_storage"), nil, nil, nil, true, pulumi.DependsOn([]pulumi.Resource{poolDefineReady}))
	if err != nil {
		return []pulumi.Resource{}, err
	}
	poolStartReady, err := runner.Command("start-libvirt-pool", pulumi.String("virsh pool-start cluster_storage"), nil, nil, nil, true, pulumi.DependsOn([]pulumi.Resource{poolBuildReady}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	poolRefreshDone, err := runner.Command("refresh-libvirt-pool", pulumi.String("virsh pool-refresh cluster_storage"), nil, nil, nil, true, pulumi.DependsOn([]pulumi.Resource{poolStartReady}))
	if err != nil {
		return []pulumi.Resource{}, err
	}
	volXmlWritten, err := runner.Command("write-vol-xml", pulumi.String(fmt.Sprintf("echo \"%s\" > /tmp/vol.xml", volXml)), nil, nil, nil, false, pulumi.DependsOn([]pulumi.Resource{libvirtReady}))
	if err != nil {
		return []pulumi.Resource{}, err
	}
	baseVolumeReady, err := runner.Command("build-libvirt-basevolume", pulumi.String("virsh vol-create cluster_storage /tmp/vol.xml"), nil, nil, nil, true, pulumi.DependsOn([]pulumi.Resource{poolRefreshDone, extractRootfs, volXmlWritten}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	uploadImageToVolumeReady, err := runner.Command("upload-libvirt-volume", pulumi.String("virsh vol-upload /pool/cluster_storage/bullseye-base /tmp/bullseye.qcow2 --pool cluster_storage"), nil, nil, nil, true, pulumi.DependsOn([]pulumi.Resource{baseVolumeReady}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	privKey, pubKey, err := generateSSHKeyPair()
	if err != nil {
		return []pulumi.Resource{}, err
	}

	libvirtPrivateKey, err = writeKeyToTempFile(privKey, "libvirt_rsa")
	if err != nil {
		return []pulumi.Resource{}, err
	}

	sshWrite, err := runner.Command("write-ssh-key", pulumi.String(fmt.Sprintf("sudo echo \"%s\" >> ~/.ssh/authorized_keys", string(pubKey))), nil, nil, nil, false, pulumi.DependsOn([]pulumi.Resource{vm}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	return []pulumi.Resource{sshWrite, uploadImageToVolumeReady, downloadKernel}, nil

}

var xlst = `
<?xml version="1.0"?>
<xsl:stylesheet version="1.0"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output omit-xml-declaration="yes" indent="yes"/>
  <xsl:template match="node()|@*">
      <xsl:copy>
         <xsl:apply-templates select="node()|@*"/>
      </xsl:copy>
   </xsl:template>

  <xsl:template match="/domain/devices">
    <xsl:copy>
        <xsl:apply-templates select="node()|@*"/>
            <xsl:element name ="controller">
                <xsl:attribute name="type">usb</xsl:attribute>
            	<xsl:attribute name="model">
                    <xsl:value-of select="'none'"/>
            	</xsl:attribute>
            </xsl:element>
    </xsl:copy>
  </xsl:template>
  <xsl:template match="domain/devices/graphics"/>
  <xsl:template match="domain/devices/audio"/>
  <xsl:template match="domain/devices/video"/>
</xsl:stylesheet>
`

//var xlst = `
//<xsl:stylesheet version="1.0"
// xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
// <xsl:output omit-xml-declaration="yes"/>
//    <xsl:template match="node()|@*">
//      <xsl:copy>
//         <xsl:apply-templates select="node()|@*"/>
//      </xsl:copy>
//    </xsl:template>
//
//    <xsl:template match="/domain/devices/controller[@type='usb']">
//            <xsl:attribute name="model">
//                    <xsl:value-of select="'none'"/>
//            </xsl:attribute>
//    </xsl:template>
//    <xsl:template match="domain/devices/graphics"/>
//    <xsl:template match="domain/devices/audio"/>
//    <xsl:template match="domain/devices/video"/>
//</xsl:stylesheet>
//`

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
	//pool, err := libvirt.NewPool(ctx, "cluster", &libvirt.PoolArgs{
	//	Type: pulumi.String("dir"),
	//	Path: pulumi.String("/pool/cluster_storage"),
	//}, pulumi.Provider(provider))
	//if err != nil {
	//	return err
	//}

	// create a filesystem volume for our VM
	// This filesystem will be based on the `ubuntu` volume above
	// we'll use a size of 10GB
	//filesystem, err := libvirt.NewVolume(ctx, "filesystem", &libvirt.VolumeArgs{
	//	Pool:   pool.Name,
	//	Source: pulumi.String("https://dd-agent-omnibus.s3.amazonaws.com/bullseye.qcow2.test"),
	//	Format: pulumi.String("qcow2"),
	//}, pulumi.Provider(provider))
	//if err != nil {
	//	return err
	//}

	filesystem, err := libvirt.NewVolume(ctx, "filesystem", &libvirt.VolumeArgs{
		BaseVolumeId: pulumi.String("/pool/cluster_storage/bullseye-base"),
		Pool:         pulumi.String("cluster_storage"),
		Format:       pulumi.String("qcow2"),
	}, pulumi.Provider(provider), pulumi.DependsOn(waitForList))
	if err != nil {
		return err
	}

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
		Kernel: pulumi.String("/tmp/bzImage"),
		Cmdlines: pulumi.MapArray{
			pulumi.Map{"console": pulumi.String("ttyS0")},
			pulumi.Map{"acpi": pulumi.String("off")},
			pulumi.Map{"panic": pulumi.String("-1")},
			pulumi.Map{"root": pulumi.String("/dev/vda")},
			pulumi.Map{"net.ifnames": pulumi.String("0")},
			pulumi.Map{"_": pulumi.String("rw")},
		},
		Memory: pulumi.Int(4096),
		Vcpu:   pulumi.Int(4),
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
	cfg, err := vmconfig.LoadFile("test.json")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%v\n", cfg)

	return

	//	pulumi.Run(func(ctx *pulumi.Context) error {
	//		e, err := aws.AWSEnvironment(ctx)
	//		if err != nil {
	//			return err
	//		}
	//
	//		instance, conn, err := newMetalInstance(e, ctx.Stack())
	//		e.Ctx.Export("instance-ip", instance.PrivateIp)
	//		e.Ctx.Export("connection", conn)
	//
	//		waitFor, err := provisionInstance(instance, conn, e)
	//		//_, err = provisionInstance(instance, conn, e)
	//		if err != nil {
	//			return nil
	//		}
	//
	//		url := pulumi.Sprintf("qemu+ssh://ubuntu@%s/system?sshauth=privkey&keyfile=%s&known_hosts_verify=ignore", instance.PrivateIp, libvirtPrivateKey)
	//		setupLibvirtVM(ctx, url, waitFor)
	//
	//		return nil
	//	})
}
