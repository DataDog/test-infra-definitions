package microvms

import (
	_ "embed"
	"fmt"
	"time"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	commonConfig "github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/namer"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/components/datadog/agent"
	"github.com/DataDog/test-infra-definitions/components/datadog/agentparams"
	"github.com/DataDog/test-infra-definitions/components/os"
	remoteComp "github.com/DataDog/test-infra-definitions/components/remote"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/resources/aws/ec2"
	ec2Scn "github.com/DataDog/test-infra-definitions/scenarios/aws/ec2"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/microVMs/config"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/microVMs/vmconfig"
)

type InstanceEnvironment struct {
	*commonConfig.CommonEnvironment
	*aws.Environment
}

type Instance struct {
	e             *InstanceEnvironment
	instance      *remoteComp.Host
	Arch          string
	instanceNamer namer.Namer
	runner        *Runner
	libvirtURI    pulumi.StringOutput
}

type sshKeyPair struct {
	privateKey string
	publicKey  string
}

const (
	LocalVMSet              = "local"
	defaultShutdownPeriod   = 360 // minutes
	libvirtSSHPrivateKeyX86 = "libvirt_rsa-x86"
	libvirtSSHPrivateKeyArm = "libvirt_rsa-arm"
)

//go:embed files/datadog.yaml
var datadogAgentConfig string

//go:embed files/system-probe.yaml
var systemProbeConfig string

var SSHKeyFileNames = map[string]string{
	ec2.AMD64Arch: libvirtSSHPrivateKeyX86,
	ec2.ARM64Arch: libvirtSSHPrivateKeyArm,
}

var GetWorkingDirectory func() string

func getKernelVersionTestingWorkingDir(m *config.DDMicroVMConfig) func() string {
	return func() string {
		return m.GetStringWithDefault(m.MicroVMConfig, config.DDMicroVMWorkingDirectory, "/tmp")
	}
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

func newMetalInstance(instanceEnv *InstanceEnvironment, name, arch string, m config.DDMicroVMConfig) (*Instance, error) {
	var instanceType string
	var ami string

	awsEnv := instanceEnv.Environment
	if awsEnv == nil {
		panic("no aws environment setup")
	}

	namer := namer.NewNamer(awsEnv.Ctx, fmt.Sprintf("%s-%s", awsEnv.Ctx.Stack(), arch))
	if arch == ec2.AMD64Arch {
		instanceType = awsEnv.DefaultInstanceType()
		ami = m.GetStringWithDefault(m.MicroVMConfig, config.DDMicroVMX86AmiID, "")
	} else if arch == ec2.ARM64Arch {
		instanceType = awsEnv.DefaultARMInstanceType()
		ami = m.GetStringWithDefault(m.MicroVMConfig, config.DDMicroVMArm64AmiID, "")
	} else {
		return nil, fmt.Errorf("unsupported arch: %s", arch)
	}

	awsInstance, err := ec2Scn.NewVM(*awsEnv, name, ec2Scn.WithAMI(ami, os.UbuntuDefault, os.Architecture(arch)), ec2Scn.WithInstanceType(instanceType))
	if err != nil {
		return nil, err
	}

	// Deploy an agent on the launched instance.
	// In the context of KMT, this agent runs on the host environment. As such,
	// it has no knowledge of the individual test VMs, other than as processes in the host machine.
	if awsEnv.AgentDeploy() {
		_, err := agent.NewHostAgent(awsEnv.CommonEnvironment, awsInstance, agentparams.WithAgentConfig(datadogAgentConfig), agentparams.WithSystemProbeConfig(systemProbeConfig))
		if err != nil {
			awsEnv.Ctx.Log.Warn(fmt.Sprintf("failed to deploy datadog agent on host instance: %v", err), nil)
		}
	}

	return &Instance{
		e:             instanceEnv,
		instance:      awsInstance,
		Arch:          arch,
		instanceNamer: namer,
	}, nil
}

func newInstance(instanceEnv *InstanceEnvironment, arch string, m config.DDMicroVMConfig) (*Instance, error) {
	name := instanceEnv.Ctx.Stack() + "-" + arch
	if arch != LocalVMSet {
		return newMetalInstance(instanceEnv, name, arch, m)
	}

	namer := namer.NewNamer(instanceEnv.Ctx, fmt.Sprintf("%s-%s", instanceEnv.Ctx.Stack(), arch))
	return &Instance{
		e:             instanceEnv,
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

func setShutdownTimer(instance *Instance, m *config.DDMicroVMConfig) (pulumi.Resource, error) {
	var shutdownRegisterDone pulumi.Resource
	shutdownPeriod := time.Duration(m.GetIntWithDefault(m.MicroVMConfig, config.DDMicroVMShutdownPeriod, defaultShutdownPeriod)) * time.Minute
	shutdownRegisterArgs := command.Args{
		Create: pulumi.Sprintf(
			"shutdown -P +%.0f", shutdownPeriod.Minutes(),
		),
		Sudo: true,
	}
	shutdownRegisterDone, err := instance.runner.Command(instance.instanceNamer.ResourceName("shutdown"), &shutdownRegisterArgs)
	if err != nil {
		return shutdownRegisterDone, fmt.Errorf("failed to schedule shutdown: %w", err)
	}

	return shutdownRegisterDone, nil
}

func configureInstance(instance *Instance, m *config.DDMicroVMConfig) ([]pulumi.Resource, error) {
	var waitFor []pulumi.Resource
	var url pulumi.StringOutput
	var err error

	env := *instance.e.CommonEnvironment
	osCommand := command.NewUnixOSCommand()
	localRunner := command.NewLocalRunner(env, command.LocalRunnerArgs{
		OSCommand: osCommand,
	})
	if instance.Arch != LocalVMSet {
		instance.runner = NewRunner(WithRemoteRunner(instance.instance.OS.Runner()))
	} else {
		instance.runner = NewRunner(WithLocalRunner(localRunner))
	}

	shouldProvision := m.GetBoolWithDefault(m.MicroVMConfig, config.DDMicroVMProvisionEC2Instance, true)
	if shouldProvision && instance.Arch != LocalVMSet {
		waitProvision, err := provisionMetalInstance(instance)
		if err != nil {
			return nil, err
		}

		waitFor = append(waitFor, waitProvision...)
	}

	if instance.Arch != LocalVMSet {
		pair := getSSHKeyPairFiles(m, instance.Arch)
		prepareSSHKeysDone, err := prepareLibvirtSSHKeys(
			instance.runner,
			localRunner,
			instance.instanceNamer,
			pair,
			nil,
		)
		if err != nil {
			return nil, err
		}
		waitFor = append(waitFor, prepareSSHKeysDone...)

		privkey := m.GetStringWithDefault(
			m.MicroVMConfig,
			config.SSHKeyConfigNames[instance.Arch],
			defaultLibvirtSSHKey(SSHKeyFileNames[instance.Arch]),
		)
		url = pulumi.Sprintf(
			"qemu+ssh://ubuntu@%s/system?sshauth=privkey&keyfile=%s&known_hosts_verify=ignore",
			instance.instance.Address,
			privkey,
		)

		if instance.e.DefaultShutdownBehavior() == "terminate" {
			shutdownTimerDone, err := setShutdownTimer(instance, m)
			if err != nil {
				return nil, err
			}
			waitFor = append(waitFor, shutdownTimerDone)
		}
	} else {
		url = pulumi.Sprintf("qemu:///system")
	}

	instance.libvirtURI = url

	return waitFor, err
}

func run(e commonConfig.CommonEnvironment) (*ScenarioDone, error) {
	var waitFor []pulumi.Resource
	var scenarioReady ScenarioDone

	m := config.NewMicroVMConfig(e)
	cfg, err := vmconfig.LoadConfigFile(
		m.GetStringWithDefault(m.MicroVMConfig, config.DDMicroVMConfigFile, "./test.json"),
	)
	if err != nil {
		return nil, err
	}

	GetWorkingDirectory = getKernelVersionTestingWorkingDir(&m)

	archs := make(map[string]bool)
	for _, set := range cfg.VMSets {
		if _, ok := archs[set.Arch]; ok {
			continue
		}
		archs[set.Arch] = true
	}

	instanceEnv := &InstanceEnvironment{&e, nil}
	// We only setup an AWS environment if we need to launch
	// a remote ec2 instance. This is determined by whether there
	// is a non-local vmset in the configuration file. The following
	// loop checks for this.
	for arch := range archs {
		if arch != LocalVMSet {
			awsEnv, err := aws.NewEnvironment(instanceEnv.Ctx, aws.WithCommonEnvironment(&e))
			if err != nil {
				return nil, err
			}
			instanceEnv.Environment = &awsEnv
			break
		}
	}

	instances := make(map[string]*Instance)
	for arch := range archs {
		instance, err := newInstance(instanceEnv, arch, m)
		if err != nil {
			return nil, err
		}

		instances[arch] = instance
	}

	for _, instance := range instances {
		configureDone, err := configureInstance(instance, &m)
		if err != nil {
			return nil, fmt.Errorf("failed to configure instance: %w", err)
		}
		scenarioReady.Instances = append(scenarioReady.Instances, instance)

		if instance.Arch != LocalVMSet {
			instanceEnv.Ctx.Export(fmt.Sprintf("%s-instance-ip", instance.Arch), instance.instance.Address)
		}

		waitFor = append(waitFor, configureDone...)
	}

	vmCollections, waitFor, err := BuildVMCollections(instances, cfg.VMSets, waitFor)
	if err != nil {
		return nil, err
	}
	scenarioReady.Dependencies, err = LaunchVMCollections(vmCollections, waitFor)
	if err != nil {
		return nil, err
	}

	// populate microVM IP mapping
	for _, collection := range vmCollections {
		for _, domain := range collection.domains {
			instanceEnv.Ctx.Export(domain.domainID, pulumi.String(domain.ip))
		}
	}

	// provision microVMs
	for _, collection := range vmCollections {
		if collection.instance.Arch == LocalVMSet {
			continue
		}

		sshConfigDone, err := setupMicroVMSSHConfig(collection.instance, microVMGroupSubnet, waitFor)
		if err != nil {
			return nil, err
		}

		microVMSSHKey, readKeyDone, err := readMicroVMSSHKey(collection.instance, sshConfigDone)
		if err != nil {
			return nil, err
		}

		for _, domain := range collection.domains {
			if domain.lvDomain == nil {
				continue
			}

			// create new ssh connection to build proxy
			conn, err := remoteComp.MakeConnection(collection.instance.instance.Address, "ubuntu", instanceEnv.DefaultPrivateKeyPath(), instanceEnv.DefaultPrivateKeyPassword(), "")
			if err != nil {
				return nil, err
			}

			pc := createProxyConnection(pulumi.String(domain.ip), "root", microVMSSHKey, conn.ToConnectionOutput())
			remoteRunner, err := command.NewRunner(
				*collection.instance.e.CommonEnvironment,
				command.RunnerArgs{
					ParentResource: domain.lvDomain,
					Connection:     pc,
					ConnectionName: collection.instance.instanceNamer.ResourceName("conn", domain.ip),
					OSCommand:      command.NewUnixOSCommand(),
				},
			)
			if err != nil {
				return nil, err
			}
			microRunner := NewRunner(WithRemoteRunner(remoteRunner))

			allowEnvDone, err := setupSSHAllowEnv(microRunner, append(readKeyDone, domain.lvDomain))
			if err != nil {
				return nil, err
			}
			_, err = reloadSSHD(microRunner, allowEnvDone)
			if err != nil {
				return nil, err
			}

			mountDisksDone, err := mountMicroVMDisks(microRunner, domain.Disks, domain.domainNamer, []pulumi.Resource{domain.lvDomain})
			if err != nil {
				return nil, err
			}

			_, err = setDockerDataRoot(microRunner, domain.Disks, domain.domainNamer, mountDisksDone)
			if err != nil {
				return nil, err
			}
		}
	}
	return &scenarioReady, nil
}

func RunAndReturnInstances(e commonConfig.CommonEnvironment) (*ScenarioDone, error) {
	return run(e)
}

func Run(ctx *pulumi.Context) error {
	commonEnv, err := commonConfig.NewCommonEnvironment(ctx)
	if err != nil {
		return err
	}

	_, err = run(commonEnv)
	return err
}
