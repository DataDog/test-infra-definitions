package utils

import (
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	// We retry every 5s maximum 5 minutes
	dialTimeoutSeconds int = 5
	dialErrorLimit     int = 60
)

func ConfigureRemoteSSH(user, sshKeyPath, sshKeyPassword, sshAgentPath string, conn *remote.ConnectionArgs) error {
	conn.PerDialTimeout = pulumi.IntPtr(dialTimeoutSeconds)
	conn.DialErrorLimit = pulumi.IntPtr(dialErrorLimit)

	conn.User = pulumi.StringPtr(user)

	if sshKeyPath != "" {
		privateKey, err := ReadSecretFile(sshKeyPath)
		if err != nil {
			return err
		}

		conn.PrivateKey = privateKey
	}

	if sshKeyPassword != "" {
		conn.PrivateKeyPassword = pulumi.StringPtr(sshKeyPassword)
	}

	if sshAgentPath != "" {
		conn.AgentSocketPath = pulumi.StringPtr(sshAgentPath)
	}

	return nil
}
