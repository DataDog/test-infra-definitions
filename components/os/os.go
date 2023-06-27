package os

import "github.com/DataDog/test-infra-definitions/components/command"

type Architecture string

const (
	AMD64Arch = Architecture("x86_64")
	ARM64Arch = Architecture("arm64")
)

type Repository string

const (
	ProdRepository    = Repository("prod")
	StagingRepository = Repository("staging")
	TestingRepository = Repository("testing")
	TrialRepository   = Repository("trial")
)

func AllowedRepositories() []Repository {
	return []Repository{ProdRepository, StagingRepository, TestingRepository, TrialRepository}
}

const (
	StableChannel  = "stable"
	BetaChannel    = "beta"
	NightlyChannel = "nightly"
)

// The types of OSes that are common
type Type int

const (
	UnixType    Type = iota
	WindowsType Type = iota
	OtherType   Type = iota
)

type AgentVersion struct {
	Major      string
	Minor      string     // Empty means latest
	Repository Repository // Defaults to prod
	Channel    string     // Defaults to stable
	PipelineID int        // Used instead of Channel when targeting the testing repository
}

func LatestAgentVersion() AgentVersion {
	return AgentVersion{
		Major:      "7",
		Repository: ProdRepository,
		Channel:    StableChannel,
	}
}

type OS interface {
	GetImage(Architecture) (string, error)
	GetDefaultInstanceType(Architecture) string
	GetServiceManager() *ServiceManager
	GetAgentConfigFolder() string
	GetSSHUser() string
	GetAgentInstallCmd(AgentVersion) (string, error)
	GetRunAgentCmd(parameters string) string
	GetType() Type
	CreatePackageManager(runner *command.Runner) (command.PackageManager, error)
}
