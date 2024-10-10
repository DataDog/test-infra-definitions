param(
    [Parameter(Mandatory,HelpMessage='authorizedkey for ssh private key for the user')]
    $authorizedKey
)

$service = Get-Service -Name sshd -ErrorAction SilentlyContinue

if ($service -ne $null) {
  Write-Host "Stop sshd service"
  Stop-Service sshd
} 

if (-not (Get-WindowsCapability -Online -Name OpenSSH.Server~~~~0.0.1.0  | Where-Object { $_.State -eq 'Installed' })) {
  Write-Host "Add OpenSSH Server capability"
  Add-WindowsCapability -Online -Name OpenSSH.Server~~~~0.0.1.0
} else {
  Write-Host "OpenSSH Server capability already installed"
}
Set-Service -Name sshd -StartupType Automatic
if (Test-Path $env:ProgramData\ssh\administrators_authorized_keys) { 
  Write-Host "Remove existing administrators_authorized_keys file"
  Remove-Item $env:ProgramData\ssh\administrators_authorized_keys
}
New-Item -Path $env:ProgramData\ssh -Name administrators_authorized_keys -ItemType file
Add-Content -Path $env:ProgramData\ssh\administrators_authorized_keys -Value $authorizedKey
icacls.exe ""$env:ProgramData\ssh\administrators_authorized_keys"" /inheritance:r /grant ""Administrators:F"" /grant ""SYSTEM:F""
New-ItemProperty -Path "HKLM:\SOFTWARE\OpenSSH" -Name DefaultShell -Value "C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe" -PropertyType String -Force
Start-Service sshd
Write-Host "OpenSSH configured."