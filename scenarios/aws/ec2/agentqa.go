package ec2

import (
	"github.com/DataDog/test-infra-definitions/common"
	"github.com/DataDog/test-infra-definitions/components/activedirectory"
	"github.com/DataDog/test-infra-definitions/components/datadog/agent"
	"github.com/DataDog/test-infra-definitions/components/datadog/agentparams"
	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/components/remote"
	"github.com/DataDog/test-infra-definitions/resources/aws"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type AgentQAClient struct {
	HostName     string
	OsDesc       *os.Descriptor
	Channel      agentparams.Channel
	MajorVersion string
	// TODO: User creds / need to create user?
}

type AgentQAOptions struct {
	Clients    []AgentQAClient
	ADDomain   string
	ADPassword string
	// Admin credentials for the domain
	ADUser         string
	ADUserPassword string
}

type AgentQAOption = func(*AgentQAOptions) error

func WithClient(client AgentQAClient) AgentQAOption {
	return func(o *AgentQAOptions) error {
		o.Clients = append(o.Clients, client)
		return nil
	}
}

func WithADOptions(adDomain string, adPassword string, adUser string, adUserPassword string) AgentQAOption {
	return func(o *AgentQAOptions) error {
		o.ADDomain = adDomain
		o.ADPassword = adPassword
		o.ADUser = adUser
		o.ADUserPassword = adUserPassword
		return nil
	}
}

type agentQAContext struct {
	pulumiContext *pulumi.Context
	env           aws.Environment
}

type windowsNodeOptions struct {
	name         string
	osDesc       os.Descriptor
	installAgent bool
	agentOptions []agentparams.Option
}

// Will create the Agent QA Environment
func AgentQARun(ctx *pulumi.Context, opts ...AgentQAOption) error {
	params, err := common.ApplyOption(&AgentQAOptions{
		ADDomain:       "ad.datadogqa.lab",
		ADPassword:     "Test1234",
		ADUser:         "ddagentuser",
		ADUserPassword: "Test5678",
	}, opts)
	if err != nil {
		return err
	}

	env, err := aws.NewEnvironment(ctx)
	if err != nil {
		return err
	}

	qaContext := &agentQAContext{
		pulumiContext: ctx,
		env:           env,
	}

	// Domain controller node
	dcForest, err := newWindowsNode(qaContext, windowsNodeOptions{name: "dcforest", installAgent: false, osDesc: os.WindowsServer2022})
	if err != nil {
		return err
	}
	_, dcForestResource, err := activedirectory.NewActiveDirectory(ctx, &env, dcForest,
		activedirectory.WithDomainController(params.ADDomain, params.ADPassword),
		activedirectory.WithDomainAdmin(params.ADUser, params.ADUserPassword),
	)
	if err != nil {
		return err
	}

	// Domain controller backup node
	dcBackup, err := newWindowsNode(qaContext, windowsNodeOptions{name: "dcbackup", installAgent: false, osDesc: os.WindowsServer2022})
	if err != nil {
		return err
	}
	_, dcBackupResource, err := activedirectory.NewActiveDirectory(ctx, &env, dcBackup,
		activedirectory.WithPulumiResourceOptions(pulumi.DependsOn(dcForestResource)),
		activedirectory.WithBackupDomainController(params.ADDomain, params.ADPassword, params.ADUser, params.ADUserPassword, dcForest),
	)
	if err != nil {
		return err
	}

	// Client nodes
	for _, client := range params.Clients {
		client, err := newWindowsNode(qaContext, windowsNodeOptions{name: client.HostName, installAgent: true, osDesc: *client.OsDesc, agentOptions: []agentparams.Option{
			agentparams.WithLatestChannel(client.Channel, client.MajorVersion),
		}})
		if err != nil {
			return err
		}
		// Setup active directory
		_, _, err = activedirectory.NewActiveDirectory(ctx, &env, client,
			// TODO: Users...
			activedirectory.WithDomain(dcForest, params.ADDomain, params.ADUser, params.ADUserPassword),
			activedirectory.WithPulumiResourceOptions(pulumi.DependsOn(dcBackupResource)),
		)
		if err != nil {
			return err
		}
	}

	return nil
}

// A client node is a host VM where the agent is installed
func newWindowsNode(qa *agentQAContext, options windowsNodeOptions) (*remote.Host, error) {
	vm, err := NewVM(qa.env, options.name, WithAMI(qa.env.InfraOSImageID(), options.osDesc, options.osDesc.Architecture), WithAgentQA())
	if err != nil {
		return nil, err
	}
	if err := vm.Export(qa.pulumiContext, nil); err != nil {
		return nil, err
	}

	if options.installAgent {
		agent, err := agent.NewHostAgent(&qa.env, vm, options.agentOptions...)
		if err != nil {
			return nil, err
		}

		err = agent.Export(qa.pulumiContext, nil)
		if err != nil {
			return nil, err
		}
	}

	return vm, nil
}
