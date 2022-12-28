package microVMs

import (
	"path/filepath"

	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/aws/ec2/ec2"
	"github.com/DataDog/test-infra-definitions/aws/scenarios/microVMs/config"
	"github.com/DataDog/test-infra-definitions/aws/scenarios/microVMs/vmconfig"
	"github.com/DataDog/test-infra-definitions/command"
	awsEc2 "github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	ddMicroVMConfigFile = "microVMConfigFile"
)

func newMetalInstance(e aws.Environment, name string) (*awsEc2.Instance, remote.ConnectionOutput, error) {
	awsInstance, conn, err := ec2.NewDefaultEC2Instance(e, name, e.DefaultInstanceType())
	if err != nil {
		return nil, remote.ConnectionOutput{}, err
	}

	return awsInstance, conn, err
}

func Run(ctx *pulumi.Context) error {
	e, err := aws.AWSEnvironment(ctx)
	if err != nil {
		return err
	}

	m := config.NewMicroVMConfig(ctx)
	cfg, err := vmconfig.LoadConfigFile(m.GetStringWithDefault(m.MicroVMConfig, ddMicroVMConfigFile, "./test.json"))
	if err != nil {
		return err
	}

	instance, conn, err := newMetalInstance(e, ctx.Stack())
	runner, err := command.NewRunner(*e.CommonEnvironment, e.Ctx.Stack()+"-conn", conn, func(r *command.Runner) (*remote.Command, error) {
		return command.WaitForCloudInit(e.Ctx, r)
	})
	localRunner := command.NewLocalRunner(*e.CommonEnvironment)

	waitFor, err := provisionInstance(runner, localRunner, &m)
	if err != nil {
		return nil
	}

	privkey := filepath.Join(m.GetStringWithDefault(m.MicroVMConfig, "tempDir", "/tmp"), libvirtSSHPrivateKey)
	url := pulumi.Sprintf("qemu+ssh://ubuntu@%s/system?sshauth=privkey&keyfile=%s&known_hosts_verify=ignore", instance.PrivateIp, privkey)
	waitForFs := []pulumi.Resource{}
	for _, set := range cfg.VMSets {
		fs := NewLibvirtFS(set.Name, &set.Img)
		d, err := fs.setupLibvirtFilesystem(runner, waitFor)
		if err != nil {
			return err
		}
		waitForFs = append(waitForFs, d...)
	}

	for _, set := range cfg.VMSets {
		setupLibvirtVM(ctx, runner, url, &set, waitForFs)
	}

	e.Ctx.Export("instance-ip", instance.PrivateIp)
	e.Ctx.Export("connection", conn)

	return nil
}
