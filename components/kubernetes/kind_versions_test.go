package kubernetes

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetKindVersionConfig(t *testing.T) {
	t.Run("existing version", func(t *testing.T) {
		config, err := GetKindVersionConfig("1.32.0")
		require.NoError(t, err)
		assert.Equal(t, "v0.26.0", config.KindVersion)
		assert.Contains(t, config.NodeImageVersion, "v1.32.0@sha256:")
	})

	t.Run("latest version", func(t *testing.T) {
		config, err := GetKindVersionConfig("latest")
		require.NoError(t, err)
		
		// Should return a valid Kind version
		assert.Regexp(t, regexp.MustCompile(`^v\d+\.\d+\.\d+$`), config.KindVersion)
		
		// Should return a version with SHA digest
		assert.Regexp(t, regexp.MustCompile(`^v\d+\.\d+\.\d+@sha256:[a-f0-9]{64}$`), config.NodeImageVersion)
		
		// Latest should be higher than any hardcoded version
		assert.True(t, config.NodeImageVersion >= "v1.32.0", "Latest version should be >= v1.32.0")
		
		t.Logf("Latest Kubernetes version: %s", config.NodeImageVersion)
	})

	t.Run("invalid version", func(t *testing.T) {
		_, err := GetKindVersionConfig("invalid")
		assert.Error(t, err)
	})

	t.Run("unsupported version", func(t *testing.T) {
		_, err := GetKindVersionConfig("999.999.999")
		assert.Error(t, err)
	})
}

func TestGetLatestKindVersion(t *testing.T) {
	config, err := getLatestKindVersion()
	require.NoError(t, err)
	
	// Should return a valid Kind version
	assert.Regexp(t, regexp.MustCompile(`^v\d+\.\d+\.\d+$`), config.KindVersion)
	
	// Should return a Kubernetes version with SHA digest
	assert.Regexp(t, regexp.MustCompile(`^v\d+\.\d+\.\d+@sha256:[a-f0-9]{64}$`), config.NodeImageVersion)
	
	t.Logf("Fetched latest: %s with Kind version: %s", config.NodeImageVersion, config.KindVersion)
}