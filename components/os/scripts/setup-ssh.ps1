$service = Get-Service -Name sshd -ErrorAction SilentlyContinue

if ($service -ne $null) {
  Write-Host "Stop sshd service"
  Stop-Service sshd
} else {
  Write-Host "sshd service not found, installing OpenSSH Server"
  # Add-WindowsCapability does NOT install a consistent version across Windows versions, this lead to
  # compatibility issues (different command line quoting rules).
  # Prefer installing sshd via MSI  
  $res = start-process -passthru -wait msiexec.exe -args '/i https://github.com/PowerShell/Win32-OpenSSH/releases/download/v9.5.0.0p1-Beta/OpenSSH-Win64-v9.5.0.0.msi /qn'
  if ($res.ExitCode -ne 0) {
    throw "SSH install failed: $($res.ExitCode)"
  }
  Write-Host "OpenSSH Server installed"
  $retries = 0
  # Confirm the Firewall rule is configured. It should be created automatically by setup. Run the following to verify
  while (!(Get-NetFirewallRule -Name "OpenSSH-Server-In-TCP" -ErrorAction SilentlyContinue).Enabled) {
    if ($retries -ge 10) {
      throw "Firewall rule 'OpenSSH-Server-In-TCP' not found after 10 retries"
    }
    if ($retries -gt 0) {
      Start-Sleep -Seconds 5
    }
    Write-Output "Firewall Rule 'OpenSSH-Server-In-TCP' does not exist, creating it..."
    New-NetFirewallRule -Name 'OpenSSH-Server-In-TCP' -DisplayName 'OpenSSH Server (sshd)' -Enabled True -Direction Inbound -Protocol TCP -Action Allow -LocalPort 22
    $retries++
  } 
  Write-Output "Firewall rule 'OpenSSH-Server-In-TCP' created."
  $powershellPath = "C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe"
  $retries = 0
  $res = Get-ItemProperty "HKLM:\SOFTWARE\OpenSSH"
  while ((Get-ItemProperty "HKLM:\SOFTWARE\OpenSSH").DefaultShell -ne $powershellPath) {
    if ($retries -ge 10) {
      throw "Failed to set powershell as default shell for sshd after 10 retries"
    }
    if ($retries -gt 0) {
      Start-Sleep -Seconds 5
    }
    Write-Host "Set powershell as default shell for sshd"
    New-ItemProperty -Path "HKLM:\SOFTWARE\OpenSSH" -Name DefaultShell -Value $powershellPath -PropertyType String -Force 
    $retries++
  }
  $retries = 0
  while (((Get-Service -Name sshd -ErrorAction SilentlyContinue) -eq $null) -and ($waitLeft -gt 0)) {
    if ($retries -ge 10) {
      throw "Failed to find sshd service after 10 retries, 5 seconds interval"
    }
    Write-Host "Waiting for sshd service to exist"
    Start-Sleep -Seconds 5
    $retries++
  }
  $retries = 0
  while ((Get-Service -Name sshd -ErrorAction SilentlyContinue).StartType -ne "Automatic") {
    if ($retries -ge 10) {
      throw "Failed to set sshd service to start automatically after 10 retries"
    }
    if ($retries -gt 0) {
      Start-Sleep -Seconds 5
    }
    Write-Host "Set sshd service to start automatically"
    Set-Service -Name sshd -StartupType Automatic
    $retries++
  }
}

Write-Host "Resetting ssh authorized keys"
$retries = 0
while (Test-Path $env:ProgramData\ssh\administrators_authorized_keys) { 
  if ($retries -ge 10) {
    throw "Failed to remove existing administrators_authorized_keys file after 10 retries"
  }
  if ($retries -gt 0) {
    Start-Sleep -Seconds 1
  }
  Write-Host "Remove existing administrators_authorized_keys file"
  Remove-Item $env:ProgramData\ssh\administrators_authorized_keys
  $retries++
}

$retries = 0
while (!Test-Path $env:ProgramData\ssh\administrators_authorized_keys) { 
  if ($retries -ge 10) {
    throw "Failed to create administrators_authorized_keys file after 10 retries"
  }
  if ($retries -gt 0) {
    Start-Sleep -Seconds 1
  }
  Write-Host "Creating administrators_authorized_keys file"
  New-Item -Path $env:ProgramData\ssh -Name administrators_authorized_keys -ItemType file
  $retries++
}
Add-Content -Path $env:ProgramData\ssh\administrators_authorized_keys -Value $authorizedKey
icacls.exe ""$env:ProgramData\ssh\administrators_authorized_keys"" /inheritance:r /grant ""Administrators:F"" /grant ""SYSTEM:F""
# Start sshd service
$retries = 0
while ((Get-Service -Name sshd -ErrorAction SilentlyContinue).Status -ne "Running") {
  if ($retries -ge 10) {
    throw "Failed to start sshd service after 10 retries"
  }
  if ($retries -gt 0) {
    Start-Sleep -Seconds 5
  }
  Write-Host "Starting sshd service"
  Start-Service sshd
  $retries++
}