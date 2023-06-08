package os

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetWindowsRepositoryURL(t *testing.T) {
	tests := []struct {
		name     string
		version  AgentVersion
		expected string
	}{
		{
			name:     "Default parameters",
			version:  LatestAgentVersion(),
			expected: "https://ddagent-windows-stable.s3.amazonaws.com",
		},
		{
			name:     "Agent 6, prod/stable",
			version:  AgentVersion{Major: "6", Repository: ProdRepository, Channel: StableChannel},
			expected: "https://ddagent-windows-stable.s3.amazonaws.com",
		},
		{
			name:     "Agent 7, prod/stable",
			version:  AgentVersion{Major: "7", Repository: ProdRepository, Channel: StableChannel},
			expected: "https://ddagent-windows-stable.s3.amazonaws.com",
		},
		{
			name:     "Agent 7, prod/beta",
			version:  AgentVersion{Major: "7", Repository: ProdRepository, Channel: BetaChannel},
			expected: "https://ddagent-windows-stable.s3.amazonaws.com/beta",
		},
		{
			name:     "Agent 7, prod/nightly",
			version:  AgentVersion{Major: "7", Repository: ProdRepository, Channel: NightlyChannel},
			expected: "https://ddagent-windows-stable.s3.amazonaws.com/nightly",
		},
		{
			name:     "Agent 7, staging/stable",
			version:  AgentVersion{Major: "7", Repository: StagingRepository, Channel: StableChannel},
			expected: "https://dd-agent-mstesting.s3.amazonaws.com/builds/stable",
		},
		{
			name:     "Agent 7, staging/beta",
			version:  AgentVersion{Major: "7", Repository: StagingRepository, Channel: BetaChannel},
			expected: "https://dd-agent-mstesting.s3.amazonaws.com/builds/beta",
		},
		{
			name:     "Agent 7, staging/nightly",
			version:  AgentVersion{Major: "7", Repository: StagingRepository, Channel: NightlyChannel},
			expected: "https://dd-agent-mstesting.s3.amazonaws.com/builds/nightly",
		},
		{
			name:     "Agent 7, trial/stable",
			version:  AgentVersion{Major: "7", Repository: TrialRepository, Channel: StableChannel},
			expected: "https://ddagent-windows-trial.s3.amazonaws.com",
		},
		{
			name:     "Agent 7, trial/beta",
			version:  AgentVersion{Major: "7", Repository: TrialRepository, Channel: BetaChannel},
			expected: "https://ddagent-windows-trial.s3.amazonaws.com/beta",
		},
		{
			name:     "Agent 7, trial/nightly",
			version:  AgentVersion{Major: "7", Repository: TrialRepository, Channel: NightlyChannel},
			expected: "https://ddagent-windows-trial.s3.amazonaws.com/nightly",
		},
	}

	for _, testInstance := range tests {
		t.Run(testInstance.name, func(t *testing.T) {
			res := getWindowsRepositoryURL(testInstance.version)
			assert.Equal(t, testInstance.expected, res)
		})
	}
}
