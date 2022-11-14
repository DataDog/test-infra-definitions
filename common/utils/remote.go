package utils

import (
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func ConfigureRemoteSSH(user, sshKeyPath, sshKeyPassword, sshAgentPath string, conn *remote.ConnectionArgs) error {
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
