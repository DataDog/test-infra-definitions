package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"

	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/aws/ec2/ec2"
	"github.com/DataDog/test-infra-definitions/aws/scenarios/micro-vms/config"
	"github.com/DataDog/test-infra-definitions/aws/scenarios/micro-vms/vmconfig"
	"github.com/DataDog/test-infra-definitions/command"
	awsEc2 "github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi-libvirt/sdk/go/libvirt"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"golang.org/x/crypto/ssh"
)

const (
	ddMicroVMConfigFile = "microVMConfigFile"
)

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
	awsInstance, conn, err := ec2.NewDefaultEC2Instance(e, name, e.DefaultInstanceType())
	if err != nil {
		return nil, remote.ConnectionOutput{}, err
	}

	return awsInstance, conn, err
}

func setupLibvirtVM(ctx *pulumi.Context, libvirtUri pulumi.StringOutput, vmset vmconfig.VMSet, waitForList []pulumi.Resource) error {
	// create a provider, this isn't required, but will make it easier to configure
	// a libvirt_uri, which we'll discuss in a bit
	provider, err := libvirt.NewProvider(ctx, "provider", &libvirt.ProviderArgs{
		Uri: libvirtUri,
	}, pulumi.DependsOn(waitForList))
	if err != nil {
		return err
	}

	domainXls, err := os.ReadFile("resources/domain.xls")
	if err != nil {
		return err
	}

	for _, kernel := range vmset.Kernels {
		network, err := libvirt.NewNetwork(ctx, "network", &libvirt.NetworkArgs{
			Addresses: pulumi.StringArray{pulumi.String("169.254.0.2/24")},
			Mode:      pulumi.String("nat"),
		}, pulumi.Provider(provider))
		if err != nil {
			return err
		}

		baseVolumeId := generatePoolPath(vmset.Name) + basefsName
		filesystem, err := libvirt.NewVolume(ctx, "filesystem", &libvirt.VolumeArgs{
			BaseVolumeId: pulumi.String(baseVolumeId),
			Pool:         pulumi.String(vmset.Name),
			Format:       pulumi.String("qcow2"),
		}, pulumi.Provider(provider), pulumi.DependsOn(waitForList))
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
			Kernel: pulumi.String(kernel.Dir),
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
				Xslt: pulumi.String(string(domainXls)),
			},
			// delete existing VM before creating replacement to avoid two VMs trying to use the same volume
		}, pulumi.Provider(provider), pulumi.ReplaceOnChanges([]string{"*"}), pulumi.DeleteBeforeReplace(true))
		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		e, err := aws.AWSEnvironment(ctx)
		if err != nil {
			return err
		}

		m := config.NewMicroVMConfig(ctx)
		cfg, err := vmconfig.LoadFile(m.GetStringWithDefault(m.MicroVMConfig, ddMicroVMConfigFile, "./test.json"))
		if err != nil {
			return err
		}

		instance, conn, err := newMetalInstance(e, ctx.Stack())
		e.Ctx.Export("instance-ip", instance.PrivateIp)
		e.Ctx.Export("connection", conn)

		runner, err := command.NewRunner(*e.CommonEnvironment, e.Ctx.Stack()+"-conn", conn, func(r *command.Runner) (*remote.Command, error) {
			return command.WaitForCloudInit(e.Ctx, r)
		})

		waitFor, err := provisionInstance(runner)
		if err != nil {
			return nil
		}

		url := pulumi.Sprintf("qemu+ssh://ubuntu@%s/system?sshauth=privkey&keyfile=%s&known_hosts_verify=ignore", instance.PrivateIp, LibvirtPrivateKey)
		waitForFs := []pulumi.Resource{}
		for _, set := range cfg.VMSets {
			d, err := setupLibvirtFilesystem(set, runner, waitFor)
			if err != nil {
				return err
			}
			waitForFs = append(waitForFs, d...)
		}

		for _, set := range cfg.VMSets {
			setupLibvirtVM(ctx, url, set, waitForFs)
		}

		return nil
	})
}
