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

func TestInvokes(t *testing.T) {
	// Arrange
	t.Log("Creating temporary configuration file")
	tmpConfigFile, err := createTemporaryConfigurationFile()
	require.NoError(t, err, "Error writing temporary configuration")
	defer os.Remove(tmpConfigFile)

	t.Log("setup test infra")
	err = setupTestInfra(tmpConfigFile)
	require.NoError(t, err)

	// Subtests
	t.Run("invoke-vm", func(t *testing.T) {
		testInvokeVM(t, tmpConfigFile)
	})
	t.Run("invoke-docker-vm", func(t *testing.T) {
		testInvokeDockerVM(t, tmpConfigFile)
	})
}

func testInvokeVM(t *testing.T, tmpConfigFile string) {
	t.Helper()
	stackName := fmt.Sprintf("invoke-vm-%s", os.Getenv("CI_PIPELINE_ID"))
	t.Log("creating vm")
	createCmd := exec.Command("invoke", "create-vm", "--no-interactive", "--stack-name", stackName, "--no-use-aws-vault", "--config-path", tmpConfigFile)
	createOutput, err := createCmd.Output()
	assert.NoError(t, err, "Error found creating vm: %s", string(createOutput))

	t.Log("destroying vm")
	destroyCmd := exec.Command("invoke", "destroy-vm", "--yes", "--no-clean-known-hosts", "--stack-name", stackName, "--no-use-aws-vault", "--config-path", tmpConfigFile)
	destroyOutput, err := destroyCmd.Output()
	require.NoError(t, err, "Error found destroying stack: %s", string(destroyOutput))
}

func testInvokeDockerVM(t *testing.T, tmpConfigFile string) {
	t.Helper()
	stackName := fmt.Sprintf("invoke-docker-vm-%s", os.Getenv("CI_PIPELINE_ID"))
	t.Log("creating vm with docker")
	var cmdStdOut, cmdStdErr bytes.Buffer

	createCmd := exec.Command("invoke", "create-docker", "--no-interactive", "--stack-name", stackName, "--no-use-aws-vault", "--config-path", tmpConfigFile)
	createCmd.Stdout = &cmdStdOut
	createCmd.Stderr = &cmdStdErr
	err := createCmd.Run()
	assert.NoError(t, err, "Error found creating docker vm.\n   stdout: %s\n  stderr: %s", cmdStdOut.String(), cmdStdErr.String())

	t.Log("destroying vm with docker")
	destroyCmd := exec.Command("invoke", "destroy-docker", "--yes", "--no-clean-known-hosts", "--stack-name", stackName, "--no-use-aws-vault", "--config-path", tmpConfigFile)
	destroyOutput, err := destroyCmd.Output()
	require.NoError(t, err, "Error found destroying stack: %s", string(destroyOutput))
}

//go:embed testfixture/config.yaml
var testInfraTestConfig string

func createTemporaryConfigurationFile() (string, error) {
	tmpConfigFile := filepath.Join(os.TempDir(), "test-infra-test.yaml")
	testInfraTestConfig = strings.ReplaceAll(testInfraTestConfig, "KEY_PAIR_NAME", os.Getenv("E2E_KEY_PAIR_NAME"))
	testInfraTestConfig = strings.ReplaceAll(testInfraTestConfig, "PUBLIC_KEY_PATH", os.Getenv("E2E_PUBLIC_KEY_PATH"))
	err := os.WriteFile(tmpConfigFile, []byte(testInfraTestConfig), 0644)
	return tmpConfigFile, err
}

func setupTestInfra(tmpConfigFile string) error {
	var setupStdout, setupStderr bytes.Buffer

	setupCmd := exec.Command("invoke", "setup", "--no-interactive", "--config-path", tmpConfigFile)
	setupCmd.Stdout = &setupStdout
	setupCmd.Stderr = &setupStderr

	setupCmd.Dir = "../"
	err := setupCmd.Run()
	if err != nil {
		return fmt.Errorf("stdout: %s\n%s, %v", setupStdout.String(), setupStderr.String(), err)
	}
	return nil
}
