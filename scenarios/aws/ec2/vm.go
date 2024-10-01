package ec2

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/components/remote"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/resources/aws/ec2"

	goremote "github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi-random/sdk/v4/go/random"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// NewVM creates an EC2 Instance and returns a Remote component.
// Without any parameter it creates an Ubuntu VM on AMD64 architecture.
func NewVM(e aws.Environment, name string, params ...VMOption) (*remote.Host, error) {
	vmArgs, err := buildArgs(params...)
	if err != nil {
		return nil, err
	}

	// Default missing parameters
	if err = defaultVMArgs(e, vmArgs); err != nil {
		return nil, err
	}

	// Resolve AMI if necessary
	amiInfo, err := resolveOS(e, vmArgs)
	if err != nil {
		return nil, err
	}

	sshUser := amiInfo.defaultUser
	if infraSSHUser := e.InfraSSHUser(); infraSSHUser != "" {
		sshUser = infraSSHUser
	}

	// Create the EC2 instance
	return components.NewComponent(&e, e.Namer.ResourceName(name), func(c *remote.Host) error {
		instanceArgs := ec2.InstanceArgs{
			AMI:             amiInfo.id,
			InstanceType:    vmArgs.instanceType,
			UserData:        vmArgs.userData,
			InstanceProfile: vmArgs.instanceProfile,
		}

		// Create the EC2 instance
		instance, err := ec2.NewInstance(e, name, instanceArgs, pulumi.Parent(c))
		if err != nil {
			return err
		}

		// Create connection
		conn, err := remote.NewConnection(instance.PrivateIp, sshUser, e.DefaultPrivateKeyPath(), e.DefaultPrivateKeyPassword(), "")
		if err != nil {
			return err
		}

		password := pulumi.String("").ToStringOutput()
		if vmArgs.osInfo.Family() == os.WindowsFamily {
			randomPassword, err := random.NewRandomString(e.Ctx(), e.Namer.ResourceName(name, "win-admin-password"), &random.RandomStringArgs{
				Length:  pulumi.Int(20),
				Special: pulumi.Bool(false),
			}, pulumi.Parent(c), e.WithProviders(config.ProviderRandom))
			if err != nil {
				return err
			}
			password = randomPassword.Result
		}

		err = remote.InitHost(&e, conn.ToConnectionOutput(), *vmArgs.osInfo, sshUser, password, amiInfo.readyFunc, c)

		if err != nil {
			return err
		}

		// reset the windows password on Windows
		if vmArgs.osInfo.Family() == os.WindowsFamily {
			_, err = c.OS.Runner().Command(
				e.CommonNamer().ResourceName("reset-admin-password"),
				&command.Args{
					Create: pulumi.Sprintf("$Password = ConvertTo-SecureString -String \"%s\" -AsPlainText -Force; Get-LocalUser -Name \"Administrator\" | Set-LocalUser -Password $Password", c.Password),
				}, pulumi.Parent(c))

			if err != nil {
				return err
			}
		}

		return nil
	})
}

func InstallECRCredentialsHelper(e aws.Environment, vm *remote.Host) (*goremote.Command, error) {
	ecrCredsHelperInstall, err := vm.OS.PackageManager().Ensure("amazon-ecr-credential-helper", nil, "")
	if err != nil {
		return nil, err
	}

	ecrConfigCommand, err := vm.OS.Runner().Command(
		e.CommonNamer().ResourceName("ecr-config"),
		&command.Args{
			Create: pulumi.Sprintf("mkdir -p ~/.docker && echo '{\"credsStore\": \"ecr-login\"}' > ~/.docker/config.json"),
			Sudo:   false,
		}, utils.PulumiDependsOn(ecrCredsHelperInstall))
	if err != nil {
		return nil, err
	}

	return ecrConfigCommand, nil
}

func defaultVMArgs(e aws.Environment, vmArgs *vmArgs) error {
	if vmArgs.osInfo == nil {
		vmArgs.osInfo = &os.UbuntuDefault
	}

	if vmArgs.instanceProfile == "" {
		vmArgs.instanceProfile = e.DefaultInstanceProfileName()
	}

	if vmArgs.instanceType == "" {
		vmArgs.instanceType = e.DefaultInstanceType()
		if vmArgs.osInfo.Architecture == os.ARM64Arch {
			vmArgs.instanceType = e.DefaultARMInstanceType()
		}
	}

	// Handle custom user data
	if vmArgs.osInfo.Family() == os.WindowsFamily {
		sshUserData, err := getWindowsOpenSSHUserData(e.DefaultPublicKeyPath())
		if err != nil {
			return err
		}

		vmArgs.userData = vmArgs.userData + sshUserData
	}

	return nil
}
