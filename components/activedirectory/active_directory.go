package activedirectory

import (
	"github.com/DataDog/test-infra-definitions/common"
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/namer"
	"github.com/DataDog/test-infra-definitions/components"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/components/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumiverse/pulumi-time/sdk/go/time"
)

// Output is an object that models the output of the resource creation
// from the Component.
// See https://www.pulumi.com/docs/concepts/resources/components/#registering-component-outputs
type Output struct {
	components.JSONImporter
}

// Component is an Active Directory domain component.
// See https://www.pulumi.com/docs/concepts/resources/components/
type Component struct {
	pulumi.ResourceState
	components.Component
	namer namer.Namer
	host  *remote.Host
}

// Export registers a key and value pair with the current context's stack.
func (dc *Component) Export(ctx *pulumi.Context, out *Output) error {
	return components.Export(ctx, dc, out)
}

// A little structure to help manage state for the Active Directory component
type activeDirectoryContext struct {
	pulumiContext    *pulumi.Context
	comp             *Component
	timeProvider     *time.Provider
	createdResources []pulumi.Resource
}

// NewActiveDirectory creates a new instance of an Active Directory domain deployment
// Example usage:
//
//	activeDirectoryComp, activeDirectoryResources, err := activedirectory.NewActiveDirectory(pulumiContext, awsEnv.CommonEnvironment, host,
//		activedirectory.WithDomainController("datadogqa.lab", "Test1234#"),
//	    activedirectory.WithDomainUser("datadogqa.lab\\ddagentuser", "Test5678#"),
//	)
//	if err != nil {
//		return err
//	}
//	err = activeDirectoryComp.Export(pulumiContext, &env.ActiveDirectory.Output)
//	if err != nil {
//		return err
//	}
func NewActiveDirectory(ctx *pulumi.Context, e *config.CommonEnvironment, host *remote.Host, options ...Option) (*Component, []pulumi.Resource, error) {
	params, err := common.ApplyOption(&Configuration{}, options)
	if err != nil {
		return nil, nil, err
	}

	adCtx := activeDirectoryContext{
		pulumiContext: ctx,
	}

	domainControllerComp, err := components.NewComponent(*e, host.Name(), func(comp *Component) error {
		comp.namer = e.CommonNamer.WithPrefix(comp.Name())
		comp.host = host
		adCtx.comp = comp

		// We use the time provider multiple times so instantiate it early
		adCtx.timeProvider, err = time.NewProvider(ctx, comp.namer.ResourceName("time-provider"), &time.ProviderArgs{}, pulumi.DeletedWith(host))
		if err != nil {
			return err
		}
		adCtx.createdResources = append(adCtx.createdResources, adCtx.timeProvider)

		if params.JoinDomainParams != nil {
			err = adCtx.joinActiveDirectoryDomain(params.JoinDomainParams)
			if err != nil {
				return err
			}
		}

		if params.DomainControllerConfiguration != nil {
			err = adCtx.installDomainController(params.DomainControllerConfiguration)
			if err != nil {
				return err
			}
		}

		if len(params.DomainUsers) > 0 {
			// Create users in parallel
			var createUserResources []pulumi.Resource
			for _, user := range params.DomainUsers {
				createDomainUserCmd, err := host.OS.Runner().Command(comp.namer.ResourceName("create-domain-users", user.Username), &command.Args{
					Create: pulumi.Sprintf(`
$HashArguments = @{
	Name = '%s'
	AccountPassword = (ConvertTo-SecureString %s -AsPlainText -Force)
	Enabled = $true
}; New-ADUser @HashArguments
`, user.Username, user.Password),
				}, pulumi.DependsOn(adCtx.createdResources))
				if err != nil {
					return err
				}
				createUserResources = append(createUserResources, createDomainUserCmd)
			}
			adCtx.createdResources = append(adCtx.createdResources, createUserResources...)
		}

		return nil
	}, pulumi.Parent(host), pulumi.DeletedWith(host))
	if err != nil {
		return nil, nil, err
	}

	return domainControllerComp, adCtx.createdResources, nil
}
