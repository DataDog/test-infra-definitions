package agentparams

type channel string

const (
	StableChannel  channel = "stable"
	BetaChannel    channel = "beta"
	NightlyChannel channel = "nightly"
)

type flavor string

const (
	DefaultFlavor flavor = BaseFlavor
	BaseFlavor    flavor = "base"
	FIPSFlavor    flavor = "fips"
)

type PackageVersion struct {
	Major      string
	Minor      string // Empty means latest
	Channel    channel
	PipelineID string
	Flavor     flavor // Empty means default (base)
}
