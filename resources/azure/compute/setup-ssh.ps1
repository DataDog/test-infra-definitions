param(
    [Parameter(Mandatory,HelpMessage='authorizedkey for ssh private key for the user')]
    $authorizedKey
)

$service = Get-Service -Name sshd -ErrorAction SilentlyContinue
if ($service -ne $null) {
  Stop-Service sshd
}
Add-WindowsCapability -Online -Name OpenSSH.Server~~~~0.0.1.0
Set-Service -Name sshd -StartupType Automatic
Add-Content -Path $env:ProgramData\ssh\administrators_authorized_keys -Value $authorizedKey
icacls.exe ""$env:ProgramData\ssh\administrators_authorized_keys"" /inheritance:r /grant ""Administrators:F"" /grant ""SYSTEM:F""
New-ItemProperty -Path "HKLM:\SOFTWARE\OpenSSH" -Name DefaultShell -Value "C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe" -PropertyType String -Force
Start-Service sshd
Write-Host "OpenSSH configured."
