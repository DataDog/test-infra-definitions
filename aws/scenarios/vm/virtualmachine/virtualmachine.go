package virtualmachine

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/aws"
	awsEc2 "github.com/DataDog/test-infra-definitions/aws/ec2/ec2"
	"github.com/DataDog/test-infra-definitions/command"
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type VirtualMachine struct {
	runner *command.Runner
	params Params
}

type Params struct {
	ami          string
	arch         string
	instanceType string
	keyPair      string
	userData     string
}

func CreateEc2InstanceParams(options ...func(*Params)) *Params {
	params := &Params{
		//name: "TODO", // ctx.Stack()
		//ami: "ami-07bc9656188ad303b", // arm64 ubuntu
		arch: awsEc2.AMD64Arch,
		//		arch: awsEc2.ARM64Arch,
		instanceType: "t3.large",
		//	instanceType: "m6g.medium",
		keyPair:  "agent-ci-sandbox",
		userData: "",
	}
	for _, o := range options {
		o(params)
	}
	return params
}

//func NewEC2Instance(e aws.Environment, name, ami, arch, instanceType, keyPair, userData string) (*ec2.Instance, error) {

// type VM struct {
// 	Context     *pulumi.Context
// 	Runner      *command.Runner
// 	Environment *config.CommonEnvironment
// 	// TODO add file manager as soon as https://github.com/DataDog/test-infra-definitions/pull/9 is merged
// 	// FileManager   *command.FileManager
// 	DockerManager *command.DockerManager
// }

func NewVirtualMachine(ctx *pulumi.Context, params Params) (*VirtualMachine, error) {
	e, err := aws.AWSEnvironment(ctx)
	if err != nil {
		return nil, err
	}

	instance, err := awsEc2.NewEC2Instance(e, ctx.Stack(), params.ami, params.arch, params.instanceType, params.keyPair, params.userData)
	if err != nil {
		return nil, err
	}

	connection, err := createConnection(instance, "ubuntu", e)
	if err != nil {
		return nil, err
	}

	runner, err := command.NewRunner(*e.CommonEnvironment, ctx.Stack()+"-conn", connection, func(r *command.Runner) (*remote.Command, error) {
		return command.WaitForCloudInit(ctx, r)
	})
	if err != nil {
		return nil, err
	}

	e.Ctx.Export("instance-ip", instance.PrivateIp)
	e.Ctx.Export("connection", connection)

	return &VirtualMachine{runner: runner}, nil
}

const (
	agentInstallCommand = `DD_API_KEY=%s %s bash -c "$(curl -L https://s3.amazonaws.com/dd-agent/scripts/install_script.sh)"`
)

func NewHostInstallation(e config.CommonEnvironment, name string, runner *command.Runner, opts ...pulumi.ResourceOption) (*remote.Command, error) {
	var extraParameters string
	agentVersion, err := config.AgentSemverVersion(&e)
	if err != nil {
		e.Ctx.Log.Info("Unable to parse Agent version, using latest", nil)
	}

	if agentVersion == nil {
		extraParameters += "DD_AGENT_MAJOR_VERSION=7"
	} else {
		extraParameters += fmt.Sprintf("DD_AGENT_MAJOR_VERSION=%d DD_AGENT_MINOR_VERSION=%d", agentVersion.Major(), agentVersion.Minor())
	}

	return runner.Command(e.CommonNamer.ResourceName(name, "agent-install"), pulumi.Sprintf(agentInstallCommand, e.AgentAPIKey(), extraParameters), nil, nil, nil, false, opts...)
}

func createConnection(instance *ec2.Instance, user string, e aws.Environment) (remote.ConnectionOutput, error) {
	connection := remote.ConnectionArgs{
		Host: instance.PrivateIp,
	}
	if err := utils.ConfigureRemoteSSH(user, e.DefaultPrivateKeyPath(), e.DefaultPrivateKeyPassword(), "", &connection); err != nil {
		return remote.ConnectionOutput{}, err
	}

	return connection.ToConnectionOutput(), nil
}
