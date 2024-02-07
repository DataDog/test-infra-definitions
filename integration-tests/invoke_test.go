package tests

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testfixture/config.yaml
var testInfraTestConfig string

func TestInvokeVM(t *testing.T) {
	var setupStdout, setupStderr bytes.Buffer

	t.Log("Creating temporary configuration file")
	tmpConfigFile := filepath.Join(os.TempDir(), "test-infra-test.yaml")
	testInfraTestConfig = strings.ReplaceAll(testInfraTestConfig, "KEY_PAIR_NAME", os.Getenv("E2E_KEY_PAIR_NAME"))
	testInfraTestConfig = strings.ReplaceAll(testInfraTestConfig, "PUBLIC_KEY_PATH", os.Getenv("E2E_PUBLIC_KEY_PATH"))
	err := os.WriteFile(tmpConfigFile, []byte(testInfraTestConfig), 0644)
	require.NoError(t, err, "Error writing temporary configuration")
	defer os.Remove(tmpConfigFile)

	t.Log("setup test infra")
	setupCmd := exec.Command("invoke", "setup", "--no-interactive", "--config-path", tmpConfigFile)
	setupCmd.Stdout = &setupStdout
	setupCmd.Stderr = &setupStderr

	setupCmd.Dir = "../"
	err = setupCmd.Run()
	require.NoError(t, err, "Error found running setup.\nstdout: %s\n%s", setupStdout.String(), setupStderr.String())

	t.Log("creating vm")
	createCmd := exec.Command("invoke", "create-vm", "--no-interactive", "--stack-name", fmt.Sprintf("integration-testing-%s", os.Getenv("CI_PIPELINE_ID")), "--no-use-aws-vault", "--config-path", tmpConfigFile)
	createOutput, err := createCmd.Output()
	assert.NoError(t, err, "Error found creating vm: %s", string(createOutput))

	t.Log("destroying vm")
	destroyCmd := exec.Command("invoke", "destroy-vm", "--yes", "--no-clean-known-hosts", "--stack-name", fmt.Sprintf("integration-testing-%s", os.Getenv("CI_PIPELINE_ID")), "--no-use-aws-vault", "--config-path", tmpConfigFile)
	destroyOutput, err := destroyCmd.Output()
	require.NoError(t, err, "Error found destroying stack: %s", string(destroyOutput))
}
