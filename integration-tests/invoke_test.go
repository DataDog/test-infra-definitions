package tests

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvokeVM(t *testing.T) {

	var setupStdout, setupStderr, createStdout, createStderr, destroyStdout, destroyStderr bytes.Buffer

	setupCmd := exec.Command("invoke", "setup", "--no-copy-to-clipboard")
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
	require.Contains(t, setupStdout.String(), "Configuration file saved at", "If setup succeeded, last message should contain 'Configuration file saved at'")

	createCmd := exec.Command("invoke", "create-vm", "--stack-name", fmt.Sprintf("integration-testing-%s", os.Getenv("CI_PIPELINE_ID")), "--no-copy-to-clipboard", "--no-use-aws-vault")
	createCmd.Stdout = &createStdout
	createCmd.Stderr = &createStderr
	err = createCmd.Run()
	assert.NoError(t, err, "Error found: %s %s", createStdout.String(), createStderr.String())

	destroyCmd := exec.Command("invoke", "destroy-vm", "--yes", "--stack-name", fmt.Sprintf("integration-testing-%s", os.Getenv("CI_PIPELINE_ID")), "--no-use-aws-vault")
	destroyCmd.Stdout = &destroyStdout
	destroyCmd.Stderr = &destroyStderr
	err = destroyCmd.Run()
	require.NoError(t, err, "Error found destroying stack: %s %s", destroyStdout.String(), destroyStderr.String())
}
