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

	var setupStdout bytes.Buffer
	var createStdout bytes.Buffer
	var createStderr bytes.Buffer
	var destroyStdout bytes.Buffer
	setupCmd := exec.Command("invoke", "setup")
	setupCmd.Stdout = &setupStdout
	stdin, _ := setupCmd.StdinPipe()

	err := setupCmd.Start()
	require.NoError(t, err)

	io.WriteString(stdin, "agent-qa\n")
	io.WriteString(stdin, fmt.Sprintf("%s\n", os.Getenv("E2E_KEY_PAIR_NAME")))
	io.WriteString(stdin, "N\n")
	io.WriteString(stdin, fmt.Sprintf("%s\n", os.Getenv("E2E_PUBLIC_KEY_PATH")))
	io.WriteString(stdin, "test-ci\n")
	io.WriteString(stdin, "00000000000000000000000000000000\n")
	io.WriteString(stdin, "0000000000000000000000000000000000000000\n")
	stdin.Close()

	setupCmd.Wait()
	require.Contains(t, setupStdout.String(), "Configuration file saved at", "If setup succeeded, last message should contain 'Configuration file saved at'")

	createCmd := exec.Command("invoke", "create-vm", "--stack-name", fmt.Sprintf("integration-testing-%s", os.Getenv("CI_PIPELINE_ID")))
	createCmd.Stdout = &createStdout
	createCmd.Stderr = &createStderr
	err = createCmd.Run()
	assert.NoError(t, err, "Error found: %s %s", createStdout.String(), createStderr.String())

	destroyCmd := exec.Command("invoke", "destroy-vm", "--yes", "--stack-name", fmt.Sprintf("integration-testing-%s", os.Getenv("CI_PIPELINE_ID")))
	destroyCmd.Stdout = &destroyStdout
	err = destroyCmd.Run()
	require.NoError(t, err, "Error found destroying stack: %s", destroyStdout.String())
}
