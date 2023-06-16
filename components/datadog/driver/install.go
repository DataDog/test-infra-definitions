package driver

import (
	"fmt"
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"os"

	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/command"
	commonos "github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/components/vm"
)

var _ utils.RemoteServiceDeserializer[ClientData] = (*Installer)(nil)

type ClientData struct {
	Connection utils.Connection
}

// Installer is an installer for a Windows Driver on a virtual machine
type Installer struct {
	dependsOn   pulumi.Resource
	vm          vm.VM
	filemanager *command.FileManager
}

// NewInstaller creates a new instance of [*Installer]
func NewInstaller(absoluteMSIPath string, vm vm.VM) (*Installer, error) {
	if osType := vm.GetOS().GetType(); osType != commonos.WindowsType {
		return nil, fmt.Errorf("driver component can only be installed on Windows VMs")
	}

	if _, err := os.Stat(absoluteMSIPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("could not find MSI at %s on host machine", absoluteMSIPath)
	}

	// TODO enable unsigned drivers
	//	execute 'enable unsigned drivers' do
	//	command "bcdedit.exe /set testsigning on"
	//notifies :reboot_now, 'reboot[now]', :immediately
	//	not_if 'bcdedit.exe | findstr "testsigning" | findstr "Yes"'
	//	end

	remotePath := "c:\\dd_driver_installer.msi"
	fileManager := vm.GetFileManager()
	_, err := fileManager.CopyFile(absoluteMSIPath, remotePath)
	if err != nil {
		return nil, fmt.Errorf("error copying MSI to remote VM: %s", err)
	}

	runner := vm.GetRunner()
	env := vm.GetCommonEnvironment()
	commonNamer := env.CommonNamer

	cmd := getDriverInstallCmd(remotePath, env)
	lastCommand, err := runner.Command(
		commonNamer.ResourceName("driver-install", utils.StrHash(cmd)),
		&command.Args{
			Create: pulumi.String(cmd),
		})
	if err != nil {
		return nil, fmt.Errorf("error installing driver: %s", err)
	}

	return &Installer{dependsOn: lastCommand, vm: vm}, err
}

func getDriverInstallCmd(remotePath string, env *config.CommonEnvironment) string {
	//	windows_package 'driver test package' do
	//installer_type :msi
	//	source "#{tmp_dir}\\kitchen\\cache\\cookbooks\\driver-base\\files\\default\\tests\\ddapmtestpackage.msi"
	//	options "/log c:\\ddapm.log"
	//action :install
	//	end

	// Disable the progress as it slow downs the download.
	cmd := "$ProgressPreference = 'SilentlyContinue'"

	// Use `if ($?) { .. }` to get an error if the install fails
	cmd += fmt.Sprintf(`; if ($?) { Start-Process -Wait msiexec -ArgumentList '/qn /i %v APIKEY="%v" SITE="datadoghq.com"'}`,
		remotePath,
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

func (installer *Installer) VM() vm.VM {
	return installer.vm
}
