package agentparams

import (
	"testing"

	"github.com/DataDog/test-infra-definitions/common"
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
	t.Run("WithIntegration should correctly add conf.d/integration/conf.yaml to the path", func(t *testing.T) {
		p := &Params{
			Integrations: make(map[string]string),
			Files:        make(map[string]string),
		}
		options := []Option{WithIntegration("http_check", "some_config")}
		result, err := common.ApplyOption(p, options)
		assert.NoError(t, err)

		for filePath, content := range result.Integrations {
			assert.Contains(t, filePath, "conf.d")
			assert.Contains(t, filePath, "http_check")
			assert.Contains(t, filePath, "conf.yaml")
			assert.Equal(t, content, "some_config")
		}
	})
}
