package compute

import (
	"fmt"
	"os"
)

func getWindowsOpenSSHUserData(publicKeyPath string) (string, error) {
	publicKey, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return "", err
	}

	openSSHInstallCmd := `<powershell>
	$service = Get-Service -Name sshd -ErrorAction SilentlyContinue
	# Don't try to reinstall OpenSSH if the user uses <persist>true</persist> on UserData.
	if ($service -eq $null) {
		Add-WindowsCapability -Online -Name OpenSSH.Server~~~~0.0.1.0
		Set-Service -Name sshd -StartupType Automatic
		Add-Content -Path $env:ProgramData\ssh\administrators_authorized_keys -Value '%v'
		icacls.exe ""$env:ProgramData\ssh\administrators_authorized_keys"" /inheritance:r /grant ""Administrators:F"" /grant ""SYSTEM:F""
		New-ItemProperty -Path "HKLM:\SOFTWARE\OpenSSH" -Name DefaultShell -Value "C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe" -PropertyType String -Force
		Start-Service sshd
	}
	</powershell>`
	return fmt.Sprintf(openSSHInstallCmd, string(publicKey)), nil
}
