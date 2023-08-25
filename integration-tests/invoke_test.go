package tests

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvokeVM(t *testing.T) {

	var setupStdout, setupStderr bytes.Buffer

	tmpConfigFile := filepath.Join(os.TempDir(), "test-infra-test.yaml")

	setupCmd := exec.Command("invoke", "setup", "--no-copy-to-clipboard", "--config-path", tmpConfigFile)
	setupCmd.Stdout = &setupStdout
	setupCmd.Stderr = &setupStderr
	stdin, _ := setupCmd.StdinPipe()

	err := setupCmd.Start()
	require.NoError(t, err)

	_, err = io.WriteString(stdin, "agent-qa\n")
	require.NoError(t, err, "Failed to write user input to stdin")
	_, err = io.WriteString(stdin, fmt.Sprintf("%s\n", os.Getenv("E2E_KEY_PAIR_NAME")))
	require.NoError(t, err, "Failed to write user input to stdin")
	_, err = io.WriteString(stdin, "N\n")
	require.NoError(t, err, "Failed to write user input to stdin")
	_, err = io.WriteString(stdin, fmt.Sprintf("%s\n", os.Getenv("E2E_PUBLIC_KEY_PATH")))
	require.NoError(t, err, "Failed to write user input to stdin")
	_, err = io.WriteString(stdin, "test-ci\n")
	require.NoError(t, err, "Failed to write user input to stdin")
	_, err = io.WriteString(stdin, "00000000000000000000000000000000\n")
	require.NoError(t, err, "Failed to write user input to stdin")
	_, err = io.WriteString(stdin, "0000000000000000000000000000000000000000\n")
	require.NoError(t, err, "Failed to write user input to stdin")
	err = stdin.Close()
	require.NoError(t, err, "Failed to close stdin pipe")

	err = setupCmd.Wait()
	require.NoError(t, err, "Error found: %s %s", setupStdout.String(), setupStderr.String())
	require.Contains(t, setupStdout.String(), fmt.Sprintf("Configuration file saved at %s", tmpConfigFile), fmt.Sprintf("If setup succeeded, last message should contain 'Configuration file saved at %s'", tmpConfigFile))

	defer os.Remove(tmpConfigFile)

	createCmd := exec.Command("invoke", "create-vm", "--stack-name", fmt.Sprintf("integration-testing-%s", os.Getenv("CI_PIPELINE_ID")), "--no-copy-to-clipboard", "--no-use-aws-vault", "--config-path", tmpConfigFile)
	createOutput, err := createCmd.Output()
	assert.NoError(t, err, "Error found: %s", string(createOutput))

	destroyCmd := exec.Command("invoke", "destroy-vm", "--yes", "--stack-name", fmt.Sprintf("integration-testing-%s", os.Getenv("CI_PIPELINE_ID")), "--no-use-aws-vault", "--config-path", tmpConfigFile)
	destroyOutput, err := destroyCmd.Output()
	require.NoError(t, err, "Error found destroying stack: %s", string(destroyOutput))
}
