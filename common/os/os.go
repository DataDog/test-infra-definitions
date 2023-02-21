package os

import "github.com/DataDog/test-infra-definitions/command"

type Architecture string

const (
	AMD64Arch = Architecture("x86_64")
	ARM64Arch = Architecture("arm64")
)

// The types of OSes that are common
type Type int

const (
	UnixType    Type = iota
	WindowsType Type = iota
	OtherType   Type = iota
)

type AgentVersion struct {
	Major       string
	Minor       string // Empty means latest
	BetaChannel bool
}

type OS interface {
	GetImage(Architecture) (string, error)
	GetDefaultInstanceType(Architecture) string
	GetServiceManager() *ServiceManager
	GetAgentConfigFolder() string
	GetSSHUser() string
	GetAgentInstallCmd(AgentVersion) (string, error)
	GetType() Type
	CreatePackageManager(runner *command.Runner) (command.PackageManager, error)
}
