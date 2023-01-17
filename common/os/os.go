package os

type Architecture string

const (
	AMD64Arch = Architecture("x86_64")
	ARM64Arch = Architecture("arm64")
)

// The types of OSes that are common
type Type int

const (
	UbuntuType  Type = iota
	WindowsType Type = iota
	OtherType   Type = iota
)

type AgentVersion struct {
	Major       string
	Minor       string
	BetaChannel bool
}

type OS interface {
	GetImage(Architecture) (string, error)
	GetDefaultInstanceType(Architecture) string
	GetServiceManager() *ServiceManager
	GetAgentConfigPath() string
	GetSSHUser() string
	GetAgentInstallCmd(AgentVersion) string
	GetType() Type
}
