package microVMs

import (
	"fmt"
	"path/filepath"

	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/aws/ec2/ec2"
	"github.com/DataDog/test-infra-definitions/aws/scenarios/microVMs/config"
	"github.com/DataDog/test-infra-definitions/aws/scenarios/microVMs/vmconfig"
	"github.com/DataDog/test-infra-definitions/command"
	"github.com/DataDog/test-infra-definitions/common"
	"github.com/DataDog/test-infra-definitions/common/utils"
	awsEc2 "github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi-libvirt/sdk/go/libvirt"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	ddMicroVMConfigFile = "microVMConfigFile"
)

type Instance struct {
	ctx           *pulumi.Context
	instance      *awsEc2.Instance
	connection    remote.ConnectionOutput
	arch          string
	instanceNamer common.Namer
	remoteRunner  *command.Runner
	localRunner   *command.LocalRunner
	libvirtURI    pulumi.StringOutput
	provider      *libvirt.Provider
}

func newMetalInstance(e aws.Environment, name, arch string) (*Instance, error) {
	var instanceType string

	namer := common.NewNamer(e.Ctx, fmt.Sprintf("%s-%s", e.Ctx.Stack(), arch))
	if arch == ec2.AMD64Arch {
		instanceType = e.DefaultInstanceType()
	} else if arch == ec2.ARM64Arch {
		instanceType = e.DefaultARMInstanceType()
	} else {
		return nil, fmt.Errorf("unsupported arch: %s", arch)
	}

	awsInstance, err := ec2.NewEC2Instance(e, name, "", arch, instanceType, e.DefaultKeyPairName(), "", "default")
	if err != nil {
		return nil, err
	}

	conn := remote.ConnectionArgs{
		Host: awsInstance.PrivateIp,
	}
	if err := utils.ConfigureRemoteSSH("ubuntu", e.DefaultPrivateKeyPath(), e.DefaultPrivateKeyPassword(), "", &conn); err != nil {
		return nil, err
	}

	return &Instance{
		ctx:           e.Ctx,
		instance:      awsInstance,
		connection:    conn.ToConnectionOutput(),
		arch:          arch,
		instanceNamer: namer,
	}, nil
}

func Run(ctx *pulumi.Context) error {
	var waitFor []pulumi.Resource

	e, err := aws.AWSEnvironment(ctx)
	if err != nil {
		return err
	}

	m := config.NewMicroVMConfig(ctx)
	cfg, err := vmconfig.LoadConfigFile(m.GetStringWithDefault(m.MicroVMConfig, ddMicroVMConfigFile, "./test.json"))
	if err != nil {
		return err
	}

	archs := make(map[string]bool)
	for _, set := range cfg.VMSets {
		if _, ok := archs[set.Arch]; ok {
			continue
		}
		archs[set.Arch] = true
	}

	instances := make(map[string]*Instance)
	for arch, _ := range archs {
		instance, err := newMetalInstance(e, ctx.Stack()+"-"+arch, arch)
		if err != nil {
			return err
		}

		instance.remoteRunner, err = command.NewRunner(*e.CommonEnvironment, instance.instanceNamer.ResourceName("conn"), instance.connection, func(r *command.Runner) (*remote.Command, error) {
			return command.WaitForCloudInit(e.Ctx, r)
		})
		if err != nil {
			return err
		}
		instance.localRunner = command.NewLocalRunner(*e.CommonEnvironment)

		wait, err := provisionInstance(instance, &m)
		if err != nil {
			return nil
		}
		waitFor = append(waitFor, wait...)

		privkey := filepath.Join(m.GetStringWithDefault(m.MicroVMConfig, "tempDir", "/tmp"), fmt.Sprintf(libvirtSSHPrivateKey, instance.arch))
		url := pulumi.Sprintf("qemu+ssh://ubuntu@%s/system?sshauth=privkey&keyfile=%s&known_hosts_verify=ignore", instance.instance.PrivateIp, privkey)

		instance.libvirtURI = url

		instances[arch] = instance

		e.Ctx.Export(fmt.Sprintf("%s-instance-ip", instance.arch), instance.instance.PrivateIp)
	}

	if err := setupLibvirtVMWithRecipe(instances, cfg.VMSets, waitFor); err != nil {
		return err
	}

	return nil
}
