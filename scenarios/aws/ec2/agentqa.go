package ec2

import (
	"github.com/DataDog/test-infra-definitions/components/activedirectory"
	"github.com/DataDog/test-infra-definitions/components/datadog/agent"
	"github.com/DataDog/test-infra-definitions/components/datadog/agentparams"
	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/components/remote"
	"github.com/DataDog/test-infra-definitions/resources/aws"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type agentQAContext struct {
	pulumiContext *pulumi.Context
	env           aws.Environment
}

// Will create the Agent QA Environment
func AgentQARun(ctx *pulumi.Context) error {
	const adDomain = "ad.datadogqa.lab"
	const adPassword = "Test1234"
	const adUser = "ddagentuser"
	const adUserPassword = "Test5678"

	env, err := aws.NewEnvironment(ctx)
	if err != nil {
		return err
	}

	qaContext := &agentQAContext{
		pulumiContext: ctx,
		env:           env,
	}

	// Domain controller node
	dcForest, err := newWindowsNode(qaContext, "dcforest", false)
	if err != nil {
		return err
	}
	_, dcForestResource, err := activedirectory.NewActiveDirectory(ctx, &env, dcForest,
		activedirectory.WithDomainController(adDomain, adPassword),
		activedirectory.WithDomainAdmin(adUser, adUserPassword),
	)
	if err != nil {
		return err
	}

	// Domain controller backup node
	dcBackup, err := newWindowsNode(qaContext, "dcbackup", false)
	if err != nil {
		return err
	}
	_, dcBackupResource, err := activedirectory.NewActiveDirectory(ctx, &env, dcBackup,
		activedirectory.WithPulumiResourceOptions(pulumi.DependsOn(dcForestResource)),
		activedirectory.WithBackupDomainController(adDomain, adPassword, adUser, adUserPassword, dcForest),
	)
	if err != nil {
		return err
	}

	// Client node
	client, err := newWindowsNode(qaContext, "client", true)
	if err != nil {
		return err
	}
	// Setup active directory
	_, _, err = activedirectory.NewActiveDirectory(ctx, &env, client,
		activedirectory.WithDomain(dcForest, adDomain, adUser, adUserPassword),
		activedirectory.WithPulumiResourceOptions(pulumi.DependsOn(dcBackupResource)),
	)
	if err != nil {
		return err
	}

	return nil
}

// TODO: This will be configured outside of test-infra-definitions

// A client node is a host VM where the agent is installed
func newWindowsNode(qa *agentQAContext, name string, installAgent bool) (*remote.Host, error) {
	// TODO: Os version / image config
	osDesc := os.WindowsServer2022
	vm, err := NewVM(qa.env, name, WithAMI(qa.env.InfraOSImageID(), osDesc, osDesc.Architecture))
	if err != nil {
		return nil, err
	}
	if err := vm.Export(qa.pulumiContext, nil); err != nil {
		return nil, err
	}

	if installAgent {
		// TODO: Configure options differently for different nodes
		agentOptions := []agentparams.Option{}
		// TODO: Custom config
		// if qa.env.AgentUseFakeintake() {
		// 	fakeIntakeOptions := []fakeintake.Option{}

		// 	if qa.env.InfraShouldDeployFakeintakeWithLB() {
		// 		fakeIntakeOptions = append(fakeIntakeOptions, fakeintake.WithLoadBalancer())
		// 	}

		// 	fakeintake, err := fakeintake.NewECSFargateInstance(qa.env, vm.Name(), fakeIntakeOptions...)
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// 	agentOptions = append(agentOptions, agentparams.WithFakeintake(fakeintake))
		// }

		// if qa.env.AgentFlavor() != "" {
		// 	agentOptions = append(agentOptions, agentparams.WithFlavor(qa.env.AgentFlavor()))
		// }

		// if qa.env.AgentConfigPath() != "" {
		// 	configContent, err := qa.env.CustomAgentConfig()
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// 	agentOptions = append(agentOptions, agentparams.WithAgentConfig(configContent))
		// }

		agent, err := agent.NewHostAgent(&qa.env, vm, agentOptions...)
		if err != nil {
			return nil, err
		}

		err = agent.Export(qa.pulumiContext, nil)
		if err != nil {
			return nil, err
		}
	}

	// TODO
	// if qa.env.UpdaterDeploy() {
	// 	if qa.env.AgentDeploy() {
	// 		return nil, errors.New("cannot deploy both agent and updater installers, updater installs the agent")
	// 	}

	// 	_, err := updater.NewHostUpdater(&qa.env, vm)
	// 	return nil, err
	// }

	return vm, nil
}
