package agentparams

type channel string

const (
	StableChannel  channel = "stable"
	BetaChannel    channel = "beta"
	NightlyChannel channel = "nightly"
)

const (
	DefaultFlavor = BaseFlavor
	BaseFlavor    = "base"
	FIPSFlavor    = "fips"
)

type PackageVersion struct {
	Major      string
	Minor      string // Empty means latest
	Channel    channel
	PipelineID string
	Flavor     string // Empty means default (base)
}
