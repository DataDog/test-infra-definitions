package ec2vm

import (
	"fmt"
	"os"

	"github.com/DataDog/test-infra-definitions/aws"
	awsEc2 "github.com/DataDog/test-infra-definitions/aws/ec2"
	commonos "github.com/DataDog/test-infra-definitions/common/os"
	commonvm "github.com/DataDog/test-infra-definitions/common/vm"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type EC2VM struct {
	env aws.Environment
	commonvm.VM
}

// NewEc2VM creates a new EC2 instance. By default use WithOS(os.UbuntuOS, os.AMD64Arch).
func NewEc2VM(ctx *pulumi.Context, options ...func(*Params) error) (*EC2VM, error) {
	return newVM(ctx, options...)
}

func (vm *EC2VM) GetAwsEnvironment() aws.Environment {
	return vm.env
}

type EC2UnixVM struct {
	*commonvm.UnixVM
	*EC2VM
}

// NewUnixEc2VM creates a new EC2 instance. By default use WithOS(os.UbuntuOS, os.AMD64Arch).
// The returned vm provides additional methods compared to NewEc2VM
func NewUnixEc2VM(ctx *pulumi.Context, options ...func(*Params) error) (*EC2UnixVM, error) {
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
		EC2VM:  vm,
	}, nil
}

// newVM creates a new EC2 instance. By default use WithOS(os.UbuntuOS, os.AMD64Arch).
func newVM(ctx *pulumi.Context, options ...func(*Params) error) (*EC2VM, error) {
	env, err := aws.NewEnvironment(ctx)
	if err != nil {
		return nil, err
	}

	params, err := newParams(env, options...)
	if err != nil {
		return nil, err
	}

	os := params.common.OS
	userData := params.common.UserData
	if os.GetType() == commonos.WindowsType {
		cmd, err := GetOpenSSHInstallCmd(env.DefaultPublicKeyPath())
		if err != nil {
			return nil, err
		}
		userData += cmd
	}
	instance, err := awsEc2.NewEC2Instance(
		env,
		env.CommonNamer.ResourceName(params.common.ImageName),
		params.common.ImageName,
		os.GetAMIArch(params.common.Arch),
		params.common.InstanceType,
		env.DefaultKeyPairName(),
		userData,
		os.GetTenancy())
	if err != nil {
		return nil, err
	}

	vm, err := commonvm.NewGenericVM(
		params.common.InstanceName,
		&env,
		instance.PrivateIp,
		os,
	)
	if err != nil {
		return nil, err
	}

	return &EC2VM{
		VM:  vm,
		env: env,
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
