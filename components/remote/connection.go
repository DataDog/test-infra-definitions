package remote

import (
	"github.com/DataDog/test-infra-definitions/common/utils"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// NewConnection creates a remote connection to a host.
// Host and user are mandatory.
func NewConnection(host pulumi.StringInput, user string, options ...ConnectionOption) (*remote.ConnectionArgs, error) {
	args, err := buildConnectionArgs(host, user, options...)

	if err != nil {
		return nil, err
	}
	conn := &remote.ConnectionArgs{
		Host:           args.host,
		User:           pulumi.String(args.user),
		PerDialTimeout: pulumi.IntPtr(args.perDialTimeoutSeconds),
		DialErrorLimit: pulumi.IntPtr(args.dialErrorLimit),
		Port:           pulumi.Float64Ptr(float64(args.port)),
	}

	if args.privateKeyPath != "" {
		privateKey, err := utils.ReadSecretFile(args.privateKeyPath)
		if err != nil {
			return nil, err
		}

		conn.PrivateKey = privateKey
	}

	if args.privateKeyPassword != "" {
		conn.PrivateKeyPassword = pulumi.StringPtr(args.privateKeyPassword)
	}

	if args.sshAgentPath != "" {
		conn.AgentSocketPath = pulumi.StringPtr(args.sshAgentPath)
	}

	return conn, nil
}
