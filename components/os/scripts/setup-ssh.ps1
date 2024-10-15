$service = Get-Service -Name sshd -ErrorAction SilentlyContinue

if ($service -ne $null) {
  Write-Host "Stop sshd service"
  Stop-Service sshd
} else {
  Write-Host "sshd service not found, installing OpenSSH Server"
    # Add-WindowsCapability does NOT install a consistent version across Windows versions, this lead to
    # compatability issues (different command line quoting rules).
    # Prefer installing sshd via MSI  
    start-process -passthru -wait msiexec.exe -args '/i https://github.com/PowerShell/Win32-OpenSSH/releases/download/v9.5.0.0p1-Beta/OpenSSH-Win64-v9.5.0.0.msi /qn'
    # Confirm the Firewall rule is configured. It should be created automatically by setup. Run the following to verify
    if (!(Get-NetFirewallRule -Name "OpenSSH-Server-In-TCP" -ErrorAction SilentlyContinue | Select-Object Name, Enabled)) {
          Write-Output "Firewall Rule 'OpenSSH-Server-In-TCP' does not exist, creating it..."
          New-NetFirewallRule -Name 'OpenSSH-Server-In-TCP' -DisplayName 'OpenSSH Server (sshd)' -Enabled True -Direction Inbound -Protocol TCP -Action Allow -LocalPort 22
    } else {
          Write-Output "Firewall rule 'OpenSSH-Server-In-TCP' has been created and exists."
    }
    # Set powershell default shell
    New-ItemProperty -Path "HKLM:\SOFTWARE\OpenSSH" -Name DefaultShell -Value "C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe" -PropertyType String -Force
    # Set sshd service to start automatically
    Set-Service -Name sshd -StartupType Automatic
}


# Reset the authorized_keys file
if (Test-Path $env:ProgramData\ssh\administrators_authorized_keys) { 
  Write-Host "Remove existing administrators_authorized_keys file"
  Remove-Item $env:ProgramData\ssh\administrators_authorized_keys
}
New-Item -Path $env:ProgramData\ssh -Name administrators_authorized_keys -ItemType file
Add-Content -Path $env:ProgramData\ssh\administrators_authorized_keys -Value $authorizedKey
icacls.exe ""$env:ProgramData\ssh\administrators_authorized_keys"" /inheritance:r /grant ""Administrators:F"" /grant ""SYSTEM:F""
# Start sshd service
Start-Service sshd
Write-Host "OpenSSH configured."