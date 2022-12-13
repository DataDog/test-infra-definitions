package main

import (
	"encoding/base64"
	"fmt"

	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/aws/ec2/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	winUserData = `<powershell>
$file = $env:SystemRoot + "\Temp\" + (Get-Date).ToString("MM-dd-yy-hh-mm")
New-Item $file -ItemType file
Enable-PSRemoting -force -SkipNetworkProfileCheck
Add-WindowsCapability -Online -Name OpenSSH.Server~~~~0.0.1.0
Get-Service -Name sshd | Set-Service -StartupType Automatic
Add-Content -Force -Path $env:ProgramData\ssh\administrators_authorized_keys -Value '%s'
icacls.exe ""$env:ProgramData\ssh\administrators_authorized_keys"" /inheritance:r /grant ""Administrators:F"" /grant ""SYSTEM:F""
Restart-Service sshd
</powershell>
<persist>true</persist>`

	// paste public key here
	// TODO: replace with public key via -c ddinfra: environment variable
	pubKey = ``
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		e, err := aws.AWSEnvironment(ctx)
		if err != nil {
			return err
		}
		fmtUserData := fmt.Sprintf(winUserData, pubKey)
		userData := base64.StdEncoding.EncodeToString([]byte(fmtUserData))
		instance, err := ec2.NewEC2Instance(
			e,
			ctx.Stack(),

			// TODO: allow to be specified / configurable
			"ami-064d05b4fe8515623",

			// TODO: allow to be specified / configurable
			"x86_64",
			e.DefaultInstanceType(),
			e.DefaultKeyPairName(),
			userData)
		if err != nil {
			return err
		}
		e.Ctx.Export("instance-ip", instance.PrivateIp)
		e.Ctx.Export("host-id", instance.HostId)
		return nil
	})
}
