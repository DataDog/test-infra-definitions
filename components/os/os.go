package os

import "github.com/DataDog/test-infra-definitions/components/command"

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
	PipelineID  string
}

// Create an alias to avoid changing too many code.
type OS CloudProviderOS

type CloudProviderOS interface {
	RawOS
	GetImage(Architecture) (string, error)
	GetDefaultInstanceType(Architecture) string
	GetSSHUser() string
}

// RawOS defines the methods which doesn't depend on any cloud provider for an OS.
type RawOS interface {
	GetServiceManager() *ServiceManager
	GetAgentConfigFolder() string
	GetAgentInstallCmd(AgentVersion) (string, error)
	GetRunAgentCmd(parameters string) string
	GetType() Type
	CreatePackageManager(runner *command.Runner) (command.PackageManager, error)
	CheckIsAbsPath(string) bool
}
