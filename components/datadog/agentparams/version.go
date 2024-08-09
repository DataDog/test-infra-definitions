package agentparams

type channel string

const (
	StableChannel  channel = "stable"
	BetaChannel    channel = "beta"
	NightlyChannel channel = "nightly"
)

type PackageVersion struct {
	Major         string
	Minor         string // Empty means latest
	Channel       channel
	PipelineID    string
	CustomVersion string
}
