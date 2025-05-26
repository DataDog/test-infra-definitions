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

// Sets the primary DNS server to the domain controller.
func (adCtx *activeDirectoryContext) setPrimaryDCDns(dc *remote.Host) error {
	setDnsCmd, err := adCtx.comp.host.OS.Runner().Command(adCtx.comp.namer.ResourceName("set-dns"), &command.Args{
		Create: pulumi.Sprintf(`
$interface = (Get-NetAdapter | Select-Object -First 1).InterfaceAlias
Set-DnsClientServerAddress -InterfaceAlias $interface -ServerAddresses ("%s")
`, dc.Address),
		Delete: pulumi.Sprintf(`
$interface = (Get-NetAdapter | Select-Object -First 1).InterfaceAlias
Set-DnsClientServerAddress -InterfaceAlias $interface -ResetServerAddresses
`),
	}, pulumi.Parent(adCtx.comp), pulumi.DependsOn(adCtx.createdResources))
	if err != nil {
		return err
	}
	adCtx.createdResources = append(adCtx.createdResources, setDnsCmd)

	return nil
}

func (adCtx *activeDirectoryContext) joinActiveDirectoryDomain(params *JoinDomainConfiguration) error {
	err := adCtx.setPrimaryDCDns(params.DomainController)
	if err != nil {
		return err
	}

	var joinCmd command.Command
	joinCmd, err = adCtx.comp.host.OS.Runner().Command(adCtx.comp.namer.ResourceName("join-domain"), &command.Args{
		Create: pulumi.Sprintf(`
	# Join the domain
	try {
		Add-Computer -DomainName "%s" -Credential (New-Object System.Management.Automation.PSCredential -ArgumentList "%[1]s\%s", (ConvertTo-SecureString -String "%s" -AsPlainText -Force))
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
	}, pulumi.Parent(adCtx.comp), pulumi.DependsOn(adCtx.createdResources))
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
	// If we want to setup a backup domain controller
	IsBackup  bool
	PrimaryDC *remote.Host
	// Credentials for a user with admin rights on the domain
	// Required for backup domain controller
	AdminName     string
	AdminPassword string
}

// WithDomainController promotes the machine to be a domain controller.
func WithDomainController(domainFqdn, domainPassword string) func(*Configuration) error {
	return func(p *Configuration) error {
		p.DomainControllerConfiguration = &DomainControllerConfiguration{
			DomainName:     domainFqdn,
			DomainPassword: domainPassword,
		}
		return nil
	}
}

// WithBackupDomainController promotes the machine to be a secondary read-only domain controller.
// The admin credentials are the credentials of a user with admin rights on the domain.
func WithBackupDomainController(domainFqdn, domainPassword, adminName, adminPassword string, primaryDc *remote.Host) func(*Configuration) error {
	return func(p *Configuration) error {
		p.DomainControllerConfiguration = &DomainControllerConfiguration{
			DomainName:     domainFqdn,
			DomainPassword: domainPassword,
			AdminName:      adminName,
			AdminPassword:  adminPassword,
			PrimaryDC:      primaryDc,
			IsBackup:       true,
		}
		return nil
	}
}

func getComputerName(name string, host *remote.Host, opts ...pulumi.ResourceOption) (command.Command, error) {
	cmd, err := host.OS.Runner().Command(name, &command.Args{
		Create: pulumi.Sprintf(`Write-Host $env:COMPUTERNAME`),
	}, opts...)
	if err != nil {
		return cmd, err
	}

	return cmd, nil
}

// Note for backup domain controllers: the host must have joined the domain before being promoted to a backup domain controller.
func (adCtx *activeDirectoryContext) installDomainController(params *DomainControllerConfiguration) error {
	var err error
	var installCmd command.Command
	if params.IsBackup {
		// Setup DNS
		err := adCtx.setPrimaryDCDns(params.PrimaryDC)
		if err != nil {
			return err
		}

		primaryDCFQDNCmd, err := getComputerName(adCtx.comp.namer.ResourceName("primary-dc-fqdn"), params.PrimaryDC)
		if err != nil {
			return err
		}
		adCtx.createdResources = append(adCtx.createdResources, primaryDCFQDNCmd)
		primaryDCFQDN := pulumi.Sprintf("%s.%s", primaryDCFQDNCmd.StdoutOutput(), params.DomainName)

		// Install the backup domain controller
		installCmd, err = adCtx.comp.host.OS.Runner().Command(adCtx.comp.namer.ResourceName("install-dc-backup"), &command.Args{
			Create: pulumi.Sprintf(`
Add-WindowsFeature -name ad-domain-services -IncludeManagementTools;
Import-Module ADDSDeployment;
try {
	Get-ADDomainController
} catch {
	$HashArguments = @{
		DomainName                    = "%[1]s"
		SafeModeAdministratorPassword = (ConvertTo-SecureString "%[2]s" -AsPlainText -Force)
		Credential                    = (New-Object System.Management.Automation.PSCredential -ArgumentList "%[1]s\%[4]s", (ConvertTo-SecureString -String "%[5]s" -AsPlainText -Force))
		InstallDns                    = $true
		Force                         = $true
		SiteName                      = "Default-First-Site-Name"
		NoGlobalCatalog               = $false
		ReadOnlyReplica               = $true
		ReplicationSourceDC           = "%[6]s"
	}; Install-ADDSDomainController @HashArguments
}
`, params.DomainName, params.DomainPassword, params.PrimaryDC.Address, params.AdminName, params.AdminPassword, primaryDCFQDN),
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
		SafeModeAdministratorPassword = (ConvertTo-SecureString "%s" -AsPlainText -Force)
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
	IsAdmin  bool
}

// WithDomainUser adds a user in Active Directory.
// Note: We don't need to be a Domain Controller to create new user in AD but we need
// the necessary rights to modify the AD.
func WithDomainUser(username, password string) func(params *Configuration) error {
	return func(p *Configuration) error {
		p.DomainUsers = append(p.DomainUsers, DomainUser{
			Username: username,
			Password: password,
			IsAdmin:  false,
		})
		return nil
	}
}

func WithDomainAdmin(username, password string) func(params *Configuration) error {
	return func(p *Configuration) error {
		p.DomainUsers = append(p.DomainUsers, DomainUser{
			Username: username,
			Password: password,
			IsAdmin:  true,
		})
		return nil
	}
}
