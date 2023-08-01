package tests

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestInvokeVM(t *testing.T) {

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	setupCmd := exec.Command("invoke", "setup")
	setupCmd.WaitDelay = 5 * time.Second
	setupCmd.Stdout = &stdout
	setupCmd.Stderr = &stderr
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
	require.Contains(t, stdout.String(), "Configuration file saved at", "If setup succeeded, last message should contain 'Configuration file saved at'")

	createCmd := exec.Command("invoke", "create-vm", fmt.Sprintf("-s %s", os.Getenv("CI_PIPELINE_ID")))
	createCmd.Run()

}
