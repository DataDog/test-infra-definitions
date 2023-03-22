package microvms

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/aws/ec2"
	"github.com/DataDog/test-infra-definitions/aws/scenarios/microVMs/config"
	"github.com/DataDog/test-infra-definitions/aws/scenarios/microVMs/vmconfig"
	"github.com/DataDog/test-infra-definitions/command"
	"github.com/DataDog/test-infra-definitions/common/namer"
	"github.com/DataDog/test-infra-definitions/common/utils"
	awsEc2 "github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi-libvirt/sdk/go/libvirt"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Instance struct {
	ctx           *pulumi.Context
	instance      *awsEc2.Instance
	Connection    remote.ConnectionOutput
	Arch          string
	instanceNamer namer.Namer
	remoteRunner  *command.Runner
	localRunner   *command.LocalRunner
	libvirtURI    pulumi.StringOutput
	provider      *libvirt.Provider
}

type sshKeyPair struct {
	privateKey string
	publicKey  string
}

func getSSHKeyPairFiles(m *config.DDMicroVMConfig, arch string) sshKeyPair {
	var pair sshKeyPair
	pair.privateKey = m.GetStringWithDefault(
		m.MicroVMConfig,
		config.SSHKeyConfigNames[arch],
		defaultLibvirtSSHKey(SSHKeyFileNames[arch]),
	)
	pair.publicKey = fmt.Sprintf(
		"%s.pub",
		m.GetStringWithDefault(
			m.MicroVMConfig,
			config.SSHKeyConfigNames[arch],
			defaultLibvirtSSHKey(SSHKeyFileNames[arch]),
		),
	)

	return pair
}

func newEC2Instance(e aws.Environment, name, ami, arch, instanceType, keyPair, userData, tenancy string) (*awsEc2.Instance, error) {
	var err error
	if ami == "" {
		ami, err = ec2.LatestUbuntuAMI(e, arch)
		if err != nil {
			return nil, err
		}
	}

	instance, err := awsEc2.NewInstance(e.Ctx, e.Namer.ResourceName(name), &awsEc2.InstanceArgs{
		Ami:                 pulumi.StringPtr(ami),
		SubnetId:            pulumi.StringPtr(e.DefaultSubnets()[0]),
		InstanceType:        pulumi.StringPtr(instanceType),
		VpcSecurityGroupIds: pulumi.ToStringArray(e.DefaultSecurityGroups()),
		KeyName:             pulumi.StringPtr(keyPair),
		UserData:            pulumi.StringPtr(userData),
		Tenancy:             pulumi.StringPtr(tenancy),
		RootBlockDevice: awsEc2.InstanceRootBlockDeviceArgs{
			VolumeSize: pulumi.Int(e.DefaultInstanceStorageSize()),
		},
		Tags: pulumi.StringMap{
			"Name": e.Namer.DisplayName(pulumi.String(name)),
			"Team": pulumi.String("ebpf-platform"),
		},
		InstanceInitiatedShutdownBehavior: pulumi.String(e.DefaultShutdownBehavior()),
	}, e.ResourceProvidersOption())
	return instance, err
}

func newMetalInstance(e aws.Environment, name, arch string, m config.DDMicroVMConfig) (*Instance, error) {
	var instanceType string
	var ami string

	namer := namer.NewNamer(e.Ctx, fmt.Sprintf("%s-%s", e.Ctx.Stack(), arch))
	if arch == ec2.AMD64Arch {
		instanceType = e.DefaultInstanceType()
		ami = m.GetStringWithDefault(m.MicroVMConfig, config.DDMicroVMX86AmiID, "")
	} else if arch == ec2.ARM64Arch {
		instanceType = e.DefaultARMInstanceType()
		ami = m.GetStringWithDefault(m.MicroVMConfig, config.DDMicroVMArm64AmiID, "")
	} else {
		return nil, fmt.Errorf("unsupported arch: %s", arch)
	}

	awsInstance, err := newEC2Instance(e, name, ami, arch, instanceType, e.DefaultKeyPairName(), "", "default")
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
		Connection:    conn.ToConnectionOutput(),
		Arch:          arch,
		instanceNamer: namer,
	}, nil
}

type ScenarioDone struct {
	Instances    []*Instance
	Dependencies []pulumi.Resource
}

func defaultLibvirtSSHKey(keyname string) string {
	return "/tmp/" + keyname
}

func run(e aws.Environment) (*ScenarioDone, error) {
	var waitFor []pulumi.Resource
	var scenarioReady ScenarioDone

	m := config.NewMicroVMConfig(e)
	cfg, err := vmconfig.LoadConfigFile(m.GetStringWithDefault(m.MicroVMConfig, config.DDMicroVMConfigFile, "./test.json"))
	if err != nil {
		return nil, err
	}

	GetWorkingDirectory = getKernelVersionTestingWorkingDir(&m)
	shouldProvision := m.GetBoolWithDefault(m.MicroVMConfig, config.DDMicroVMProvisionEC2Instance, true)

	archs := make(map[string]bool)
	for _, set := range cfg.VMSets {
		if _, ok := archs[set.Arch]; ok {
			continue
		}
		archs[set.Arch] = true
	}

	instances := make(map[string]*Instance)
	for arch := range archs {
		instance, err := newMetalInstance(e, e.Ctx.Stack()+"-"+arch, arch, m)
		if err != nil {
			return nil, err
		}

		instance.remoteRunner, err = command.NewRunner(*e.CommonEnvironment, instance.instanceNamer.ResourceName("conn"), instance.Connection, func(r *command.Runner) (*remote.Command, error) {
			return command.WaitForCloudInit(r)
		}, command.WithUser("libvirt-qemu"))
		if err != nil {
			return nil, err
		}
		instance.localRunner = command.NewLocalRunner(*e.CommonEnvironment)

		if shouldProvision {
			waitProvision, err := provisionInstance(instance, &m)
			if err != nil {
				return nil, err
			}

			waitFor = append(waitFor, waitProvision...)
		}

		// prepare ssh key pair for libvirt daemon
		pair := getSSHKeyPairFiles(&m, instance.Arch)
		prepareSSHKeysDone, err := prepareLibvirtSSHKeys(
			instance.remoteRunner,
			instance.localRunner,
			instance.instanceNamer,
			pair,
			[]pulumi.Resource{},
		)
		if err != nil {
			return nil, err
		}
		waitFor = append(waitFor, prepareSSHKeysDone...)

		privkey := m.GetStringWithDefault(m.MicroVMConfig, config.SSHKeyConfigNames[arch], defaultLibvirtSSHKey(SSHKeyFileNames[arch]))
		url := pulumi.Sprintf("qemu+ssh://ubuntu@%s/system?sshauth=privkey&keyfile=%s&known_hosts_verify=ignore", instance.instance.PrivateIp, privkey)

		instance.libvirtURI = url

		instances[arch] = instance
		scenarioReady.Instances = append(scenarioReady.Instances, instance)

		e.Ctx.Export(fmt.Sprintf("%s-instance-ip", instance.Arch), instance.instance.PrivateIp)

	}

	var microVMIPMap map[string]string
	scenarioReady.Dependencies, microVMIPMap, err = setupLibvirtVMWithRecipe(instances, cfg.VMSets, waitFor)
	if err != nil {
		return nil, err
	}

	for domainID, ip := range microVMIPMap {
		e.Ctx.Export(domainID, pulumi.String(ip))
	}

	return &scenarioReady, nil
}

func RunAndReturnInstances(e aws.Environment) (*ScenarioDone, error) {
	return run(e)
}

func Run(ctx *pulumi.Context) error {
	e, err := aws.NewEnvironment(ctx)
	if err != nil {
		return err
	}

	_, err = run(e)
	return err
}
