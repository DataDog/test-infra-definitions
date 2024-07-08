package tests

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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
	t.Cleanup(func() {
		t.Log("Cleaning up temporary configuration file")
		os.Remove(tmpConfigFile)
	})

	t.Log("setup test infra")
	err = setupTestInfra(tmpConfigFile)
	require.NoError(t, err, "Error setting up test infra")

	tmpConfig, err := LoadConfig(tmpConfigFile)
	require.NoError(t, err)

	require.NotEmpty(t, tmpConfig.ConfigParams.AWS.TeamTag)

	// Subtests
	t.Run("invoke-vm", func(t *testing.T) {
		t.Parallel()
		testInvokeVM(t, tmpConfigFile)
	})
	t.Run("invoke-docker-vm", func(t *testing.T) {
		t.Parallel()
		testInvokeDockerVM(t, tmpConfigFile)
	})
	t.Run("invoke-kind", func(t *testing.T) {
		t.Parallel()
		testInvokeKind(t, tmpConfigFile)
	})
	t.Run("invoke-kind-operator", func(t *testing.T) {
	testInvokeKindOperator(t, tmpConfigFile)
	})
}

func testInvokeVM(t *testing.T, tmpConfigFile string) {
	t.Helper()

	stackName := fmt.Sprintf("invoke-vm-%s", os.Getenv("CI_PIPELINE_ID"))
	t.Log("creating vm")
	createCmd := exec.Command("invoke", "create-vm", "--no-interactive", "--stack-name", stackName, "--config-path", tmpConfigFile, "--use-fakeintake")
	createOutput, err := createCmd.Output()
	assert.NoError(t, err, "Error found creating vm: %s", string(createOutput))

	t.Log("destroying vm")
	destroyCmd := exec.Command("invoke", "destroy-vm", "--yes", "--no-clean-known-hosts", "--stack-name", stackName, "--config-path", tmpConfigFile)
	destroyOutput, err := destroyCmd.Output()
	require.NoError(t, err, "Error found destroying stack: %s", string(destroyOutput))
}

func testInvokeDockerVM(t *testing.T, tmpConfigFile string) {
	t.Helper()
	stackName := fmt.Sprintf("invoke-docker-vm-%s", os.Getenv("CI_PIPELINE_ID"))
	t.Log("creating vm with docker")
	var stdOut, stdErr bytes.Buffer

	createCmd := exec.Command("invoke", "create-docker", "--no-interactive", "--stack-name", stackName, "--config-path", tmpConfigFile, "--use-fakeintake", "--use-loadBalancer")
	createCmd.Stdout = &stdOut
	createCmd.Stderr = &stdErr
	err := createCmd.Run()
	assert.NoError(t, err, "Error found creating docker vm.\n   stdout: %s\n   stderr: %s", stdOut.String(), stdErr.String())

	stdOut.Reset()
	stdErr.Reset()

	t.Log("destroying vm with docker")
	destroyCmd := exec.Command("invoke", "destroy-docker", "--yes", "--stack-name", stackName, "--config-path", tmpConfigFile)
	destroyCmd.Stdout = &stdOut
	destroyCmd.Stderr = &stdErr
	err = destroyCmd.Run()
	require.NoError(t, err, "Error found destroying stack.\n   stdout: %s\n   stderr: %s", stdOut.String(), stdErr.String())
}

func testInvokeKind(t *testing.T, tmpConfigFile string) {
	t.Helper()
	stackParts := []string{"invoke", "kind"}
	if os.Getenv("CI") == "true" {
		stackParts = append(stackParts, os.Getenv("CI_PIPELINE_ID"))
	}
	stackName := strings.Join(stackParts, "-")
	t.Log("creating kind cluster")
	createCmd := exec.Command("invoke", "create-kind", "--no-interactive", "--stack-name", stackName, "--config-path", tmpConfigFile, "--use-fakeintake", "--use-loadBalancer")
	createOutput, err := createCmd.Output()
	assert.NoError(t, err, "Error found creating kind cluster: %s", string(createOutput))

	t.Log("destroying kind cluster")
	destroyCmd := exec.Command("invoke", "destroy-kind", "--yes", "--stack-name", stackName, "--no-use-aws-vault", "--config-path", tmpConfigFile)
	destroyOutput, err := destroyCmd.Output()
	require.NoError(t, err, "Error found destroying kind cluster: %s", string(destroyOutput))
}

func testInvokeKindOperator(t *testing.T, tmpConfigFile string) {
	t.Helper()
	stackName := "invoke-kind-agent-with-operator"
	if os.Getenv("CI") == "true" {
		stackName = fmt.Sprintf("%s-%s", stackName, os.Getenv("CI_PIPELINE_ID"))
	}

	t.Log("creating kind cluster with operator")
	createCmd := exec.Command("invoke", "create-kind", "--install-agent-with-operator", "--no-interactive", "--stack-name", stackName, "--no-use-aws-vault", "--config-path", tmpConfigFile)
	createOutput, err := createCmd.Output()
	assert.NoError(t, err, "Error found creating kind cluster: %s", string(createOutput))

	t.Log("destroying kind cluster")
	destroyCmd := exec.Command("invoke", "destroy-kind", "--yes", "--stack-name", stackName, "--config-path", tmpConfigFile)
	destroyOutput, err := destroyCmd.Output()
	require.NoError(t, err, "Error found destroying kind cluster: %s", string(destroyOutput))
}

//go:embed testfixture/config.yaml
var testInfraTestConfig string

func createTemporaryConfigurationFile() (string, error) {
	tmpConfigFile := filepath.Join(os.TempDir(), "test-infra-test.yaml")

	isCI, err := strconv.ParseBool(os.Getenv("CI"))
	account := "agent-qa"
	keyPairName := os.Getenv("E2E_KEY_PAIR_NAME")
	publicKeyPath := os.Getenv("E2E_PUBLIC_KEY_PATH")
	if err != nil || !isCI {
		// load local config
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		localConfig, err := LoadConfig(filepath.Join(homeDir, ".test_infra_config.yaml"))
		if err != nil {
			return "", err
		}
		account = localConfig.ConfigParams.AWS.Account
		keyPairName = localConfig.ConfigParams.AWS.KeyPairName
		publicKeyPath = localConfig.ConfigParams.AWS.PublicKeyPath
	}
	testInfraTestConfig = strings.ReplaceAll(testInfraTestConfig, "KEY_PAIR_NAME", keyPairName)
	testInfraTestConfig = strings.ReplaceAll(testInfraTestConfig, "PUBLIC_KEY_PATH", publicKeyPath)
	testInfraTestConfig = strings.ReplaceAll(testInfraTestConfig, "ACCOUNT", account)
	err = os.WriteFile(tmpConfigFile, []byte(testInfraTestConfig), 0644)
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
