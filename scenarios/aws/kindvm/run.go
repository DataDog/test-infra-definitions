package kindvm

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/components/datadog/agent"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/cpustress"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/dogstatsd"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/mutatedbyadmissioncontroller"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/nginx"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/prometheus"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/redis"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/tracegen"
	dogstatsdstandalone "github.com/DataDog/test-infra-definitions/components/datadog/dogstatsd-standalone"
	fakeintakeComp "github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
	"github.com/DataDog/test-infra-definitions/components/docker"
	localKubernetes "github.com/DataDog/test-infra-definitions/components/kubernetes"
	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/components/remote"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	resAws "github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/ec2"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/fakeintake"

	goremote "github.com/pulumi/pulumi-command/sdk/go/command/remote"

	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Run(ctx *pulumi.Context) error {
	awsEnv, err := resAws.NewEnvironment(ctx)
	if err != nil {
		return err
	}

	osDesc := os.DescriptorFromString(awsEnv.InfraOSDescriptor(), os.AmazonLinuxECSDefault)
	vm, err := ec2.NewVM(awsEnv, "kind", ec2.WithOS(osDesc))
	if err != nil {
		return err
	}
	if err := vm.Export(ctx, nil); err != nil {
		return err
	}

	// Install docker if not installed yet, we need it to configure docker credentials
	_, dockerInstallCmd, err := docker.NewManager(*awsEnv.CommonEnvironment, vm)
	if err != nil {
		return err
	}
	// Configure ECR credentials for use in Kind
	ecrLoginCommand, err := ConfigureECRCredentials(awsEnv, vm, osDesc.Architecture, utils.PulumiDependsOn(dockerInstallCmd))
	if err != nil {
		return err
	}

	kindClusterName := ctx.Stack()

	kindCluster, err := localKubernetes.NewKindCluster(*awsEnv.CommonEnvironment, vm, awsEnv.CommonNamer.ResourceName("kind"), kindClusterName, awsEnv.KubernetesVersion(), utils.PulumiDependsOn(ecrLoginCommand))
	if err != nil {
		return err
	}
	if err := kindCluster.Export(ctx, nil); err != nil {
		return err
	}

	// Building Kubernetes provider
	kindKubeProvider, err := kubernetes.NewProvider(ctx, awsEnv.Namer.ResourceName("k8s-provider"), &kubernetes.ProviderArgs{
		EnableServerSideApply: pulumi.BoolPtr(true),
		Kubeconfig:            kindCluster.KubeConfig,
	})
	if err != nil {
		return err
	}

	var dependsOnCrd pulumi.ResourceOption

	var fakeIntake *fakeintakeComp.Fakeintake
	if awsEnv.GetCommonEnvironment().AgentUseFakeintake() {
		fakeIntakeOptions := []fakeintake.Option{
			fakeintake.WithCPU(1024),
			fakeintake.WithMemory(6144),
		}
		if awsEnv.GetCommonEnvironment().InfraShouldDeployFakeintakeWithLB() {
			fakeIntakeOptions = append(fakeIntakeOptions, fakeintake.WithLoadBalancer())
		}

		if fakeIntake, err = fakeintake.NewECSFargateInstance(awsEnv, kindCluster.Name(), fakeIntakeOptions...); err != nil {
			return err
		}
		if err := fakeIntake.Export(awsEnv.Ctx, nil); err != nil {
			return err
		}
	}

	// Deploy the agent
	if awsEnv.AgentDeploy() {
		customValues := fmt.Sprintf(`
datadog:
  kubelet:
    tlsVerify: false
  clusterName: "%s"
agents:
  useHostNetwork: true
`, kindClusterName)

		helmComponent, err := agent.NewHelmInstallation(*awsEnv.CommonEnvironment, agent.HelmInstallationArgs{
			KubeProvider: kindKubeProvider,
			Namespace:    "datadog",
			ValuesYAML: pulumi.AssetOrArchiveArray{
				pulumi.NewStringAsset(customValues),
			},
			Fakeintake: fakeIntake,
		}, nil)
		if err != nil {
			return err
		}

		ctx.Export("agent-linux-helm-install-name", helmComponent.LinuxHelmReleaseName)
		ctx.Export("agent-linux-helm-install-status", helmComponent.LinuxHelmReleaseStatus)

		dependsOnCrd = utils.PulumiDependsOn(helmComponent)
	}

	// Deploy standalone dogstatsd
	if awsEnv.DogstatsdDeploy() {
		if _, err := dogstatsdstandalone.K8sAppDefinition(*awsEnv.CommonEnvironment, kindKubeProvider, "dogstatsd-standalone", fakeIntake, false, kindClusterName); err != nil {
			return err
		}
	}

	// Deploy testing workload
	if awsEnv.TestingWorkloadDeploy() {
		if _, err := nginx.K8sAppDefinition(*awsEnv.CommonEnvironment, kindKubeProvider, "workload-nginx", "", dependsOnCrd); err != nil {
			return err
		}

		if _, err := redis.K8sAppDefinition(*awsEnv.CommonEnvironment, kindKubeProvider, "workload-redis", dependsOnCrd); err != nil {
			return err
		}

		if _, err := cpustress.K8sAppDefinition(*awsEnv.CommonEnvironment, kindKubeProvider, "workload-cpustress"); err != nil {
			return err
		}

		// dogstatsd clients that report to the Agent
		if _, err := dogstatsd.K8sAppDefinition(*awsEnv.CommonEnvironment, kindKubeProvider, "workload-dogstatsd", 8125, "/var/run/datadog/dsd.socket"); err != nil {
			return err
		}

		// dogstatsd clients that report to the dogstatsd standalone deployment
		if _, err := dogstatsd.K8sAppDefinition(*awsEnv.CommonEnvironment, kindKubeProvider, "workload-dogstatsd-standalone", dogstatsdstandalone.HostPort, dogstatsdstandalone.Socket); err != nil {
			return err
		}

		// for tracegen we can't find the cgroup version as it depends on the underlying version of the kernel
		if _, err := tracegen.K8sAppDefinition(*awsEnv.CommonEnvironment, kindKubeProvider, "workload-tracegen"); err != nil {
			return err
		}

		if _, err := prometheus.K8sAppDefinition(*awsEnv.CommonEnvironment, kindKubeProvider, "workload-prometheus"); err != nil {
			return err
		}

		if _, err := mutatedbyadmissioncontroller.K8sAppDefinition(*awsEnv.CommonEnvironment, kindKubeProvider, "workload-mutated", "workload-mutated-lib-injection"); err != nil {
			return err
		}
	}

	return nil
}

func ConfigureECRCredentials(e aws.Environment, vm *remote.Host, arch os.Architecture, opts ...pulumi.ResourceOption) (*goremote.Command, error) {
	architecture := "x86_64"
	if arch == os.ARM64Arch {
		architecture = "aarch64"
	}

	unzipInstallCommand, err := vm.OS.PackageManager().Ensure("unzip", nil, "")
	if err != nil {
		return nil, err
	}

	awsCliInstallCommand, err := vm.OS.Runner().Command(
		e.CommonNamer.ResourceName("aws-cli-install"),
		&command.Args{
			Create: pulumi.Sprintf("command -v aws || curl 'https://awscli.amazonaws.com/awscli-exe-linux-%s.zip' -o 'awscliv2.zip' && unzip awscliv2.zip && sudo ./aws/install", architecture),
		},
		utils.PulumiDependsOn(unzipInstallCommand),
	)
	if err != nil {
		return nil, err
	}

	ecrLoginCommand, err := vm.OS.Runner().Command(
		e.CommonNamer.ResourceName("ecr-login"),
		&command.Args{
			Create: pulumi.Sprintf("aws ecr get-login-password | docker --config /tmp/kind-config login  --username AWS --password-stdin %s", e.CloudProviderEnvironment.InternalRegistry()),
		},
		utils.PulumiDependsOn(awsCliInstallCommand),
	)

	return ecrLoginCommand, err
}
