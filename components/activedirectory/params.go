package activedirectory

import (
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/components/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumiverse/pulumi-time/sdk/go/time"
)

// Configuration is an object representing the desired Active Directory configuration.
type Configuration struct {
	JoinDomainParams              *JoinDomainConfiguration
	DomainControllerConfiguration *DomainControllerConfiguration
	DomainUsers                   []DomainUser
	ResourceOptions               []pulumi.ResourceOption
}

// Option is an optional function parameter type for Configuration options
type Option = func(*Configuration) error

// WithPulumiResourceOptions sets some pulumi resource option, like which resource
// to depend on.
func WithPulumiResourceOptions(resources ...pulumi.ResourceOption) Option {
	return func(p *Configuration) error {
		p.ResourceOptions = resources
		return nil
	}
}

// JoinDomainConfiguration list the options required for a machine to join an Active Directory domain.
type JoinDomainConfiguration struct {
	DomainController        *remote.Host
	DomainName              string
	DomainAdminUser         string
	DomainAdminUserPassword string
}

// WithDomain joins a machine to a domain. The machine can then be promoted to a domain controller or remain
// a domain client.
// The domainAdmin is "mydomain.com\myuser"
func WithDomain(domainController *remote.Host, domainFqdn, domainAdmin, domainAdminPassword string) Option {
	return func(p *Configuration) error {
		p.JoinDomainParams = &JoinDomainConfiguration{
			DomainController:        domainController,
			DomainName:              domainFqdn,
			DomainAdminUser:         domainAdmin,
			DomainAdminUserPassword: domainAdminPassword,
		}
		return nil
	}
}

func (adCtx *activeDirectoryContext) joinActiveDirectoryDomain(params *JoinDomainConfiguration) error {
	// TODO: Need restart?
	setDnsCmd, err := adCtx.comp.host.OS.Runner().Command(adCtx.comp.namer.ResourceName("set-dns"), &command.Args{
		Create: pulumi.Sprintf(`
# Set the primary DNS to the domain controller
$interface = (Get-NetAdapter | Select-Object -First 1).InterfaceAlias
Set-DnsClientServerAddress -InterfaceAlias $interface -ServerAddresses ("%s")
`, params.DomainController.Address),
		Delete: pulumi.Sprintf(`
$interface = (Get-NetAdapter | Select-Object -First 1).InterfaceAlias
Set-DnsClientServerAddress -InterfaceAlias $interface -ResetServerAddresses
`),
	}, pulumi.Parent(adCtx.comp))
	adCtx.createdResources = append(adCtx.createdResources, setDnsCmd)

	var joinCmd command.Command
	joinCmd, err = adCtx.comp.host.OS.Runner().Command(adCtx.comp.namer.ResourceName("join-domain"), &command.Args{
		Create: pulumi.Sprintf(`
# Join the domain
try {
	Add-Computer -DomainName "%s" -Credential (New-Object System.Management.Automation.PSCredential -ArgumentList "%s", (ConvertTo-SecureString -String "%s" -AsPlainText -Force))
} catch {
 	if ($_.Exception.Message -like "*already in that domain*") {
		Write-Host "Already joined to domain"
	} else {
		throw $_
	}
}
`, params.DomainName, params.DomainAdminUser, params.DomainAdminUserPassword),
		// TODO: This hangs
		// 		Delete: pulumi.Sprintf(`
		// Remove-Computer -UnjoinDomainCredential %s -PassThru -Force
		// `, params.DomainAdminUser),
	}, pulumi.Parent(setDnsCmd))
	if err != nil {
		return err
	}
	adCtx.createdResources = append(adCtx.createdResources, joinCmd)

	waitForRebootAfterJoiningCmd, err := time.NewSleep(adCtx.pulumiContext, adCtx.comp.namer.ResourceName("wait-for-host-to-reboot-after-joining-domain"), &time.SleepArgs{
		CreateDuration: pulumi.String("30s"),
	},
		pulumi.Provider(adCtx.timeProvider),
		pulumi.DependsOn(adCtx.createdResources)) // Depend on all the previously created resources
	if err != nil {
		return err
	}
	adCtx.createdResources = append(adCtx.createdResources, waitForRebootAfterJoiningCmd)
	return nil
}

// DomainControllerConfiguration represents the Active Directory configuration (domain name, password, users etc...)
type DomainControllerConfiguration struct {
	DomainName     string
	DomainPassword string
	// Whether it is the secondary / read only domain controller
	IsBackup bool
}

// WithDomainController promotes the machine to be a domain controller.
func WithDomainController(domainFqdn, adminPassword string, isBackup bool) func(*Configuration) error {
	return func(p *Configuration) error {
		p.DomainControllerConfiguration = &DomainControllerConfiguration{
			DomainName:     domainFqdn,
			DomainPassword: adminPassword,
			IsBackup:       isBackup,
		}
		return nil
	}
}

func (adCtx *activeDirectoryContext) installDomainController(params *DomainControllerConfiguration) error {
	var err error
	var installCmd command.Command
	if params.IsBackup {
		// This is the secondary domain controller so we don't use ADDSForest
		installCmd, err = adCtx.comp.host.OS.Runner().Command(adCtx.comp.namer.ResourceName("install-dc-backup"), &command.Args{
			Create: pulumi.Sprintf(`
Add-WindowsFeature -name ad-domain-services -IncludeManagementTools;
Import-Module ADDSDeployment;
try {
	Get-ADDomainController
} catch {
	Install-ADDSDomainController -DomainName "%1s" -ReadOnlyReplica -Credential (New-Object System.Management.Automation.PSCredential -ArgumentList "%1s", (ConvertTo-SecureString -String "%2s" -AsPlainText -Force)) -InstallDNS -NoGlobalCatalog:$true
}
`, params.DomainName, params.DomainPassword),
		}, pulumi.Parent(adCtx.comp), pulumi.DependsOn(adCtx.createdResources))
		if err != nil {
			return err
		}
	} else {
		installCmd, err = adCtx.comp.host.OS.Runner().Command(adCtx.comp.namer.ResourceName("install-forest"), &command.Args{
			Create: pulumi.Sprintf(`
Add-WindowsFeature -name ad-domain-services -IncludeManagementTools;
Import-Module ADDSDeployment;
try {
	Get-ADDomainController
} catch {
	$HashArguments = @{
		CreateDNSDelegation           = $false
		ForestMode                    = "Win2012R2"
		DomainMode                    = "Win2012R2"
		DomainName                    = "%s"
		SafeModeAdministratorPassword = (ConvertTo-SecureString %s -AsPlainText -Force)
		Force                         = $true
	}; Install-ADDSForest @HashArguments
}
`, params.DomainName, params.DomainPassword),
		}, pulumi.Parent(adCtx.comp), pulumi.DependsOn(adCtx.createdResources))
		if err != nil {
			return err
		}
	}
	adCtx.createdResources = append(adCtx.createdResources, installCmd)

	waitForRebootCmd, err := time.NewSleep(adCtx.pulumiContext, adCtx.comp.namer.ResourceName("wait-for-host-to-reboot"), &time.SleepArgs{
		CreateDuration: pulumi.String("30s"),
	},
		pulumi.Provider(adCtx.timeProvider),
		pulumi.DependsOn(adCtx.createdResources)) // Depend on all the previously created resources
	if err != nil {
		return err
	}
	adCtx.createdResources = append(adCtx.createdResources, waitForRebootCmd)

	ensureAdwsStartedCmd, err := adCtx.comp.host.OS.Runner().Command(adCtx.comp.namer.ResourceName("ensure-adws-started"), &command.Args{
		Create: pulumi.String(`(Get-Service ADWS).WaitForStatus('Running', '00:01:00')`),
	}, utils.PulumiDependsOn(waitForRebootCmd))
	if err != nil {
		return err
	}
	adCtx.createdResources = append(adCtx.createdResources, ensureAdwsStartedCmd)
	return nil
}

// DomainUser represents an Active Directory user
type DomainUser struct {
	Username string
	Password string
}

// WithDomainUser adds a user in Active Directory.
// Note: We don't need to be a Domain Controller to create new user in AD but we need
// the necessary rights to modify the AD.
func WithDomainUser(username, password string) func(params *Configuration) error {
	return func(p *Configuration) error {
		p.DomainUsers = append(p.DomainUsers, DomainUser{
			Username: username,
			Password: password,
		})
		return nil
	}
}
