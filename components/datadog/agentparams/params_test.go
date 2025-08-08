package agentparams

import (
	"strings"
	"testing"

	"github.com/DataDog/test-infra-definitions/common"
	perms "github.com/DataDog/test-infra-definitions/components/datadog/agentparams/filepermissions"
	"github.com/stretchr/testify/assert"
)

func TestParams(t *testing.T) {
	t.Run("parseVersion should correctly parse stable version", func(t *testing.T) {
		version, err := parseVersion("7.43")
		assert.NoError(t, err)
		assert.Equal(t, version, PackageVersion{
			Major:   "7",
			Minor:   "43",
			Channel: StableChannel,
		})
	})
	t.Run("parseVersion should correctly parse rc version", func(t *testing.T) {
		version, err := parseVersion("7.45~rc.1")
		assert.NoError(t, err)
		assert.Equal(t, version, PackageVersion{
			Major:   "7",
			Minor:   "45~rc.1",
			Channel: BetaChannel,
		})
	})
	t.Run("parsePipelineVersion should correctly parse a pipeline ID and format the agent version pipeline", func(t *testing.T) {
		p := &Params{}
		options := []Option{WithPipeline("16362517")}
		result, err := common.ApplyOption(p, options)
		assert.NoError(t, err)
		assert.Equal(t, result.Version, PackageVersion{
			PipelineID: "16362517",
		})
	})
	t.Run("WithIntegration should correctly add conf.d/integration/conf.yaml to the path", func(t *testing.T) {
		p := &Params{
			Integrations: make(map[string]*FileDefinition),
			Files:        make(map[string]*FileDefinition),
		}
		options := []Option{WithIntegration("http_check", "some_config")}
		result, err := common.ApplyOption(p, options)
		assert.NoError(t, err)

		for filePath, definition := range result.Integrations {
			assert.Contains(t, filePath, "conf.d")
			assert.Contains(t, filePath, "http_check")
			assert.Contains(t, filePath, "conf.yaml")
			assert.Equal(t, definition.Content, "some_config")
		}
	})
	t.Run("WithBase64BinaryFileWithPermissions should create a base64 shell script and file entry", func(t *testing.T) {
		p := &Params{
			Integrations: make(map[string]*FileDefinition),
			Files:        make(map[string]*FileDefinition),
		}

		data := []byte("dummy binary content")
		targetPath := "/opt/bin/test-binary"
		options := []Option{
			WithBase64BinaryFileWithPermissions(
				targetPath,
				data,
				true,
				perms.NewUnixPermissions(perms.WithPermissions("0500"), perms.WithGroup("root"), perms.WithOwner("root")),
			),
		}

		result, err := common.ApplyOption(p, options)
		assert.NoError(t, err)

		// Should create two entries in Files map:
		// 1. the install script (under /tmp/)
		// 2. the actual binary path (as a placeholder)
		foundScript := false
		for path, def := range result.Files {
			if path == targetPath {
				assert.Nil(t, def.Content)
				assert.Equal(t, true, def.UseSudo)
			} else if strings.HasPrefix(path, "/tmp/install-") && strings.HasSuffix(path, ".sh") {
				foundScript = true
				assert.NotEmpty(t, def.Content)
				assert.Contains(t, def.Content, "base64 -d")
				assert.Contains(t, def.Content, "chmod 0500")
				assert.Contains(t, def.Content, targetPath)
			}
		}
		assert.True(t, foundScript, "Expected a temp install script to be registered")
	})
}
