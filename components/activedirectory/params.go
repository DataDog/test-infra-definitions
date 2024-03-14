package activedirectory

import "github.com/pulumi/pulumi/sdk/v3/go/pulumi"

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
