package os

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetUnixRepositoryParams(t *testing.T) {
	tests := []struct {
		name     string
		version  AgentVersion
		expected string
	}{
		{
			name:     "Default parameters",
			version:  LatestAgentVersion(),
			expected: "DD_REPO_URL=\"datadoghq.com\" DD_AGENT_DIST_CHANNEL=\"stable\"",
		},
		{
			name:     "Agent 6, prod/stable",
			version:  AgentVersion{Major: "6", Repository: ProdRepository, Channel: StableChannel},
			expected: "DD_REPO_URL=\"datadoghq.com\" DD_AGENT_DIST_CHANNEL=\"stable\"",
		},
		{
			name:     "Agent 7, prod/stable",
			version:  AgentVersion{Major: "7", Repository: ProdRepository, Channel: StableChannel},
			expected: "DD_REPO_URL=\"datadoghq.com\" DD_AGENT_DIST_CHANNEL=\"stable\"",
		},
		{
			name:     "Agent 7, prod/beta",
			version:  AgentVersion{Major: "7", Repository: ProdRepository, Channel: BetaChannel},
			expected: "DD_REPO_URL=\"datadoghq.com\" DD_AGENT_DIST_CHANNEL=\"beta\"",
		},
		{
			name:     "Agent 7, staging/beta",
			version:  AgentVersion{Major: "7", Repository: StagingRepository, Channel: BetaChannel},
			expected: "DD_REPO_URL=\"datad0g.com\" DD_AGENT_DIST_CHANNEL=\"beta\"",
		},
		{
			name:     "Agent 7, staging/nightly",
			version:  AgentVersion{Major: "7", Repository: StagingRepository, Channel: NightlyChannel},
			expected: "DD_REPO_URL=\"datad0g.com\" DD_AGENT_DIST_CHANNEL=\"nightly\"",
		},
		{
			name:     "Agent 6, trial/stable",
			version:  AgentVersion{Major: "6", Repository: TrialRepository, Channel: StableChannel},
			expected: "TESTING_APT_URL=\"apttrial.datad0g.com\" TESTING_APT_REPO_VERSION=\"stable 6\" TESTING_YUM_URL=\"yumtrial.datad0g.com\" TESTING_YUM_VERSION_PATH=\"stable/6\"",
		},
		{
			name:     "Agent 7, trial/stable",
			version:  AgentVersion{Major: "7", Repository: TrialRepository, Channel: StableChannel},
			expected: "TESTING_APT_URL=\"apttrial.datad0g.com\" TESTING_APT_REPO_VERSION=\"stable 7\" TESTING_YUM_URL=\"yumtrial.datad0g.com\" TESTING_YUM_VERSION_PATH=\"stable/7\"",
		},
		{
			name:     "Agent 6, testing/11111111",
			version:  AgentVersion{Major: "6", Repository: TestingRepository, PipelineID: "11111111"},
			expected: "TESTING_APT_URL=\"apttesting.datad0g.com\" TESTING_APT_REPO_VERSION=\"pipeline-11111111-a6 6\" TESTING_YUM_URL=\"yumtesting.datad0g.com\" TESTING_YUM_VERSION_PATH=\"testing/pipeline-11111111-a6/6\"",
		},
		{
			name:     "Agent 7, testing/11111111",
			version:  AgentVersion{Major: "7", Repository: TestingRepository, PipelineID: "11111111"},
			expected: "TESTING_APT_URL=\"apttesting.datad0g.com\" TESTING_APT_REPO_VERSION=\"pipeline-11111111-a7 7\" TESTING_YUM_URL=\"yumtesting.datad0g.com\" TESTING_YUM_VERSION_PATH=\"testing/pipeline-11111111-a7/7\"",
		},
		{
			name:     "Agent 7, testing/11111111, check overridden channel",
			version:  AgentVersion{Major: "7", Repository: TestingRepository, Channel: StableChannel, PipelineID: "11111111"},
			expected: "TESTING_APT_URL=\"apttesting.datad0g.com\" TESTING_APT_REPO_VERSION=\"pipeline-11111111-a7 7\" TESTING_YUM_URL=\"yumtesting.datad0g.com\" TESTING_YUM_VERSION_PATH=\"testing/pipeline-11111111-a7/7\"",
		},
	}

	for _, testInstance := range tests {
		t.Run(testInstance.name, func(t *testing.T) {
			res := getUnixRepositoryParams(testInstance.version)
			assert.Equal(t, testInstance.expected, res)
		})
	}
}
