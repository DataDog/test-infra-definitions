package activedirectory

import (
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/command"
	pulumiRemote "github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumiverse/pulumi-time/sdk/go/time"
)

// DomainControllerConfiguration represents the Active Directory configuration (domain name, password, users etc...)
type DomainControllerConfiguration struct {
	DomainName     string
	DomainPassword string
}

// WithDomainController promotes the machine to be a domain controller.
func WithDomainController(domainFqdn, adminPassword string) func(*Configuration) error {
	return func(p *Configuration) error {
		p.DomainControllerConfiguration = &DomainControllerConfiguration{
			DomainName:     domainFqdn,
			DomainPassword: adminPassword,
		}
		return nil
	}
}

func (adCtx *activeDirectoryContext) installDomainController(params *DomainControllerConfiguration) error {
	var installCmd *pulumiRemote.Command
	installCmd, err := adCtx.comp.host.OS.Runner().Command(adCtx.comp.namer.ResourceName("install-forest"), &command.Args{
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
