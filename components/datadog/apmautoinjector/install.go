package apmautoinjector

import (
	"fmt"
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"os"
	"path/filepath"

	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/command"
	commonos "github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/components/vm"
)

var _ utils.RemoteServiceDeserializer[ClientData] = (*Installer)(nil)

type ClientData struct {
	Connection utils.Connection
}

// Installer is an installer for the APM auto-injector on a virtual machine
type Installer struct {
	LastCommand pulumi.Resource
	vm          vm.VM
}

// NewInstaller creates a new instance of [*Installer]
func NewInstaller(vm vm.VM, options ...func(*Params) error) (*Installer, error) {
	if osType := vm.GetOS().GetType(); osType != commonos.WindowsType {
		return nil, fmt.Errorf("APM auto-injector component can only be installed on Windows VMs")
	}

	runner := vm.GetRunner()
	env := vm.GetCommonEnvironment()

	params, err := newParams(options...)
	if err != nil {
		return nil, err
	}

	// enable test signed drivers
	cmd := "bcdedit.exe -set TESTSIGNING ON"
	lastCommand, err := runner.Command(
		env.CommonNamer.ResourceName("enable-test-signed-drivers", utils.StrHash(cmd)),
		&command.Args{
			Create: pulumi.String(cmd),
		})
	if err != nil {
		return nil, err
	}

	// reboot for previous command to take effect
	cmd = "shutdown -r -t 0"
	lastCommand, err = runner.Command(
		env.CommonNamer.ResourceName("reboot", utils.StrHash(cmd)),
		&command.Args{
			Create: pulumi.String(cmd),
		}, utils.PulumiDependsOn(lastCommand))

	installerResource, installerPath, err := getInstaller(vm, params, lastCommand)
	if err != nil {
		return nil, err
	}

	// complete installation
	cmd = getInstallCmd(installerPath, env)
	lastCommand, err = runner.Command(
		env.CommonNamer.ResourceName("apm-auto-inject-install", utils.StrHash(cmd)),
		&command.Args{
			Create: pulumi.String(cmd),
			Delete: pulumi.String("cat c:\\ddapm.log"),
		}, utils.PulumiDependsOn(installerResource))
	if err != nil {
		return nil, fmt.Errorf("error installing APM auto-injector: %s", err)
	}

	return &Installer{LastCommand: lastCommand, vm: vm}, err
}

func getInstaller(vm vm.VM, params *Params, depends pulumi.Resource) (pulumi.Resource, string, error) {
	if params.localInstallerPath != "" {
		return copyLocalInstallerToVM(vm, params.localInstallerPath, depends)
	}
	return installLatest()
}

func copyLocalInstallerToVM(vm vm.VM, localPath string, depends pulumi.Resource) (pulumi.Resource, string, error) {
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return nil, "", fmt.Errorf("could not find %s on host machine", localPath)
	}

	installerPath := fmt.Sprintf("c:\\%s", filepath.Base(localPath), utils.PulumiDependsOn(depends))

	fileManager := vm.GetFileManager()
	resource, err := fileManager.CopyFile(localPath, installerPath)
	if err != nil {
		return nil, "", fmt.Errorf("error copying directory to remote VM: %s", err)
	}

	return resource, installerPath, nil
}

func installLatest() (pulumi.Resource, string, error) {
	// Once we have a released version of the APM auto-injector, we can fetch & install it here
	return nil, "", fmt.Errorf("APM Auto-injector component currently only supports installation via a local installer")
}

func getInstallCmd(installerPath string, env *config.CommonEnvironment) string {
	// Disable the progress as it slow downs the download.
	cmd := "$ProgressPreference = 'SilentlyContinue'"

	// Use `if ($?) { .. }` to get an error if the install fails
	cmd += fmt.Sprintf(`; if ($?) { Start-Process -Wait msiexec -ArgumentList '/qn /i %v APIKEY="%v" SITE="datadoghq.com"  /l*v c:\\ddapm.log'}`,
		installerPath,
		env.AgentAPIKey(),
	)

	return cmd
}

func (installer *Installer) Deserialize(result auto.UpResult) (*ClientData, error) {
	vmData, err := installer.vm.Deserialize(result)
	if err != nil {
		return nil, err
	}

	return &ClientData{Connection: vmData.Connection}, nil
}
