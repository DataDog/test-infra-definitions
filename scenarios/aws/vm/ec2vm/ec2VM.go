package ec2vm

import (
	"fmt"
	"os"

	componentos "github.com/DataDog/test-infra-definitions/components/os"
	commonvm "github.com/DataDog/test-infra-definitions/components/vm"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	awsEc2 "github.com/DataDog/test-infra-definitions/resources/aws/ec2"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/vm/ec2params"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Infra struct {
	env aws.Environment
}

func (infra *Infra) GetAwsEnvironment() aws.Environment {
	return infra.env
}

type EC2VM struct {
	Infra
	commonvm.VM
}

// NewEc2VM creates a new EC2 instance. By default use WithOS(os.UbuntuOS, os.AMD64Arch).
func NewEc2VM(ctx *pulumi.Context, options ...ec2params.Option) (*EC2VM, error) {
	return newVM(ctx, options...)
}

type EC2UnixVM struct {
	Infra
	*commonvm.UnixVM
}

// NewUnixEc2VM creates a new EC2 instance. By default use WithOS(os.UbuntuOS, os.AMD64Arch).
// The returned vm provides additional methods compared to NewEc2VM
func NewUnixEc2VM(ctx *pulumi.Context, options ...ec2params.Option) (*EC2UnixVM, error) {
	vm, err := newVM(ctx, options...)
	if err != nil {
		return nil, err
	}
	unixVM, err := commonvm.NewUnixVM(vm.VM)
	if err != nil {
		return nil, err
	}

	return &EC2UnixVM{
		UnixVM: unixVM,
		Infra:  vm.Infra,
	}, nil
}

// newVM creates a new EC2 instance. By default use WithOS(os.UbuntuOS, os.AMD64Arch).
func newVM(ctx *pulumi.Context, options ...ec2params.Option) (*EC2VM, error) {
	env, err := aws.NewEnvironment(ctx)
	if err != nil {
		return nil, err
	}

	params, err := ec2params.NewParams(env, options...)
	if err != nil {
		return nil, err
	}

	commonParams := params.GetCommonParams()
	osValue := commonParams.OS
	userData := commonParams.UserData
	if osValue.GetType() == componentos.WindowsType {
		cmd, err := GetOpenSSHInstallCmd(env.DefaultPublicKeyPath())
		if err != nil {
			return nil, err
		}
		userData += cmd
	}
	instance, err := awsEc2.NewEC2Instance(
		env,
		env.CommonNamer.ResourceName(commonParams.ImageName),
		commonParams.ImageName,
		osValue.GetAMIArch(commonParams.Arch),
		commonParams.InstanceType,
		env.DefaultKeyPairName(),
		userData,
		osValue.GetTenancy())
	if err != nil {
		return nil, err
	}

	vm, err := commonvm.NewGenericVM(
		commonParams.InstanceName,
		instance,
		&env,
		instance.PrivateIp,
		osValue,
	)
	if err != nil {
		return nil, err
	}

	return &EC2VM{
		VM:    vm,
		Infra: Infra{env: env},
	}, nil
}

func GetOpenSSHInstallCmd(publicKeyPath string) (string, error) {
	publicKey, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return "", err
	}

	openSSHInstallCmd := `<powershell>
	$service = Get-Service -Name sshd -ErrorAction SilentlyContinue
	# Don't try to reinstall OpenSSH if the user uses <persist>true</persist> on UserData.
	if ($service -eq $null) {
		Add-WindowsCapability -Online -Name OpenSSH.Server
		Set-Service -Name sshd -StartupType Automatic
		Add-Content -Path $env:ProgramData\ssh\administrators_authorized_keys -Value '%v'
		icacls.exe ""$env:ProgramData\ssh\administrators_authorized_keys"" /inheritance:r /grant ""Administrators:F"" /grant ""SYSTEM:F""
		New-ItemProperty -Path "HKLM:\SOFTWARE\OpenSSH" -Name DefaultShell -Value "C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe" -PropertyType String -Force
		Start-Service sshd
	}
	</powershell>`
	return fmt.Sprintf(openSSHInstallCmd, string(publicKey)), nil
}
