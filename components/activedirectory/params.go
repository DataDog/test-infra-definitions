package activedirectory

// Configuration is an object representing the desired Active Directory configuration.
type Configuration struct {
	JoinDomainParams              *JoinDomainConfiguration
	DomainControllerConfiguration *DomainControllerConfiguration
	DomainUsers                   []DomainUser
}

// Option is an optional function parameter type for Configuration options
type Option = func(*Configuration) error
