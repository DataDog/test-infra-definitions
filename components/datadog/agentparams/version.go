package agentparams

type PackageVersion struct {
	Major          string
	Minor          string // Empty means latest
	BetaChannel    bool
	NightlyChannel bool
	PipelineID     string
}
