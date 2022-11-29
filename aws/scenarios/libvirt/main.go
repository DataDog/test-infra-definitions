package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/aws/ec2/ec2"
	"github.com/DataDog/test-infra-definitions/command"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi-libvirt/sdk/go/libvirt"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func createClientCerts(pkipath string, caCertLoc string, caKeyLoc string) error {
	chain, err := tls.LoadX509KeyPair(caCertLoc, caKeyLoc)
	if err != nil {
		return err
	}

	ca, err := x509.ParseCertificate(chain.Certificate[0])
	if err != nil {
		return err
	}

	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(42),
		Subject: pkix.Name{
			Organization: []string{"Avocado"},
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(365 * 24 * time.Hour),
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature,
	}
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	pub := &priv.PublicKey

	clientCert, err := x509.CreateCertificate(rand.Reader, clientTemplate, ca, pub, chain.PrivateKey)
	if err != nil {
		return err
	}

	clientCertLoc := filepath.Join(pkipath, "clientcert.pem")
	clientKeyLoc := filepath.Join(pkipath, "clientkey.pem")

	certOut, err := os.Create(clientCertLoc)
	if err != nil {
		return err
	}

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: clientCert}); err != nil {
		return err
	}

	if err := certOut.Close(); err != nil {
		return err
	}

	keyOut, err := os.OpenFile(clientKeyLoc, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}

	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)}); err != nil {
		return err
	}

	if err := keyOut.Close(); err != nil {
		return err
	}

	return nil
}

func createCACerts(pkipath string) error {
	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(42),
		Subject: pkix.Name{
			Organization: []string{"Avocado"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(365 * 24 * time.Hour),
		IsCA:      true,
		KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
	}

	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	pub := &priv.PublicKey
	ca, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, pub, priv)
	if err != nil {
		return err
	}

	caCertLoc := filepath.Join(pkipath, "cacert.pem")
	caKeyLoc := filepath.Join(pkipath, "cakey.pem")

	certOut, err := os.Create(caCertLoc)
	if err != nil {
		return err
	}

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: ca}); err != nil {
		return err
	}

	err = certOut.Close()
	if err != nil {
		return err
	}

	keyOut, err := os.OpenFile(caKeyLoc, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}

	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)}); err != nil {
		return err
	}

	if err := keyOut.Close(); err != nil {
		return err
	}

	return createClientCerts(pkipath, caCertLoc, caKeyLoc)
}
func setupLibvirtVM(ctx *pulumi.Context, libvirtUri string) error {
	// create a provider, this isn't required, but will make it easier to configure
	// a libvirt_uri, which we'll discuss in a bit
	provider, err := libvirt.NewProvider(ctx, "provider", &libvirt.ProviderArgs{
		Uri: pulumi.String(libvirtUri),
	})
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

	// create a volume with the contents being a Ubuntu 20.04 server image
	ubuntu, err := libvirt.NewVolume(ctx, "ubuntu", &libvirt.VolumeArgs{
		Pool:   pool.Name,
		Source: pulumi.String("https://cloud-images.ubuntu.com/releases/focal/release/ubuntu-20.04-server-cloudimg-amd64.img"),
	}, pulumi.Provider(provider))
	if err != nil {
		return err
	}

	// create a filesystem volume for our VM
	// This filesystem will be based on the `ubuntu` volume above
	// we'll use a size of 10GB
	filesystem, err := libvirt.NewVolume(ctx, "filesystem", &libvirt.VolumeArgs{
		BaseVolumeId: ubuntu.ID(),
		Pool:         pool.Name,
		Size:         pulumi.Int(10000000000),
	}, pulumi.Provider(provider))
	if err != nil {
		return err
	}

	cloud_init_user_data, err := os.ReadFile("./cloud_init_user_data.yaml")
	if err != nil {
		return err
	}

	cloud_init_network_config, err := os.ReadFile("./cloud_init_network_config.yaml")
	if err != nil {
		return err
	}

	// create a cloud init disk that will setup the ubuntu credentials and enable dhcp
	cloud_init, err := libvirt.NewCloudInitDisk(ctx, "cloud-init", &libvirt.CloudInitDiskArgs{
		MetaData:      pulumi.String(string(cloud_init_user_data)),
		NetworkConfig: pulumi.String(string(cloud_init_network_config)),
		Pool:          pool.Name,
		UserData:      pulumi.String(string(cloud_init_user_data)),
	}, pulumi.Provider(provider))
	if err != nil {
		return err
	}

	// create NAT network using 192.168.10/24 CIDR
	network, err := libvirt.NewNetwork(ctx, "network", &libvirt.NetworkArgs{
		Addresses: pulumi.StringArray{pulumi.String("169.254.1.0/30")},
		Mode:      pulumi.String("nat"),
	}, pulumi.Provider(provider))
	if err != nil {
		return err
	}

	// create a VM that has a name starting with ubuntu
	_, err = libvirt.NewDomain(ctx, "ubuntu", &libvirt.DomainArgs{
		Cloudinit: cloud_init.ID(),
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

		_, err = runner.Command(e.Ctx, "restart-libvirtd", pulumi.String("sudo systemctl restart libvirtd"), nil, nil, nil, true, pulumi.DependsOn([]pulumi.Resource{disableSELinux}))
		if err != nil {
			return err
		}

		// create certs
		pkipath := os.TempDir()
		err = createCACerts(pkipath)
		if err != nil {
			return err
		}

		// explicitly set dependency?
		if err := setupLibvirtVM(e.Ctx, "qemu+ssh://ubuntu@172.29.188.86/system?sshauth=privkey&keyfile=/tmp/test_rsa&known_hosts=/home/ubuntu/.ssh/known_hosts"); err != nil {
			return err
		}

		e.Ctx.Export("instance-ip", instance.PrivateIp)
		e.Ctx.Export("connection", conn)

		return nil
	})
}
