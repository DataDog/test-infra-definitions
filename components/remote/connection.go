package remote

import (
	"github.com/DataDog/test-infra-definitions/common/utils"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	// We retry every 5s maximum 100 times (~8 minutes)
	dialTimeoutSeconds int = 5
	dialErrorLimit     int = 100
)

// NewConnection creates a remote connection to a host.
// Host and user are mandatory.
func NewConnection(host pulumi.StringInput, user, sshKeyPath, sshKeyPassword, sshAgentPath string) (*remote.ConnectionArgs, error) {
	conn := &remote.ConnectionArgs{
		Host:           host,
		User:           pulumi.String(user),
		PerDialTimeout: pulumi.IntPtr(dialTimeoutSeconds),
		DialErrorLimit: pulumi.IntPtr(dialErrorLimit),
	}

	if sshKeyPath != "" {
		privateKey, err := utils.ReadSecretFile(sshKeyPath)
		if err != nil {
			return nil, err
		}

		conn.PrivateKey = privateKey
	}

	if sshKeyPassword != "" {
		conn.PrivateKeyPassword = pulumi.StringPtr(sshKeyPassword)
	}

	if sshAgentPath != "" {
		conn.AgentSocketPath = pulumi.StringPtr(sshAgentPath)
	}

	return conn, nil
}
