package agent

import (
	"testing"

	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/stretchr/testify/assert"
)

func TestParams(t *testing.T) {
	t.Run("parseVersion should correctly parse stable version", func(t *testing.T) {
		version, err := parseVersion("7.43")
		assert.NoError(t, err)
		assert.Equal(t, version, os.AgentVersion{
			Major:       "7",
			Minor:       "43",
			BetaChannel: false,
		})
	})
	t.Run("parseVersion should correctly parse rc version", func(t *testing.T) {
		version, err := parseVersion("7.45~rc.1")
		assert.NoError(t, err)
		assert.Equal(t, version, os.AgentVersion{
			Major:       "7",
			Minor:       "45~rc.1",
			BetaChannel: true,
		})
	})
	t.Run("parsePipelineVersion should correctly parse a pipeline ID and format the agent version pipeline", func(t *testing.T) {
		version := parsePipelineVersion("16362517")
		assert.Equal(t, version, os.AgentVersion{
			PipelineID: "pipeline-16362517",
		})
	})
}
