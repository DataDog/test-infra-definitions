package agentparams

type PackageVersion struct {
	Major       string
	Minor       string // Empty means latest
	BetaChannel bool
	PipelineID  string
}
