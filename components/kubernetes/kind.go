package kubernetes

import (
	_ "embed"
	"fmt"
	"regexp"
	"strings"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/components/docker"
	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/components/remote"
)

const (
	kindReadinessWait = "60s"
	kindNodeImageName = "kindest/node"
)

//go:embed kind-cluster.yaml
var kindClusterConfig string

// Install Kind on a Linux virtual machine.
func NewKindCluster(env config.Env, vm *remote.Host, name string, kubeVersion string, opts ...pulumi.ResourceOption) (*Cluster, error) {
	return components.NewComponent(env, name, func(clusterComp *Cluster) error {
		isLocal := vm == nil
		kindClusterName := env.CommonNamer().DisplayName(49) // We can have some issues if the name is longer than 50 characters
		opts = utils.MergeOptions[pulumi.ResourceOption](opts, pulumi.Parent(clusterComp))
		kindVersionConfig, err := getKindVersionConfig(kubeVersion)
		if err != nil {
			return err
		}

		kindInstall, err := installKind(env, vm, kindVersionConfig, opts...)
		if err != nil {
			return fmt.Errorf("error installing kind: %w", err)
		}

		runner := runner(env, vm)
		fileManager := fileManager(env, runner)
		clusterConfigFilePath := fmt.Sprintf("/tmp/kind-cluster-%s.yaml", name)
		clusterConfig, err := fileManager.CopyInlineFile(
			pulumi.String(kindClusterConfig),
			clusterConfigFilePath, opts...)
		if err != nil {
			return err
		}

		dependsOnRes := []pulumi.Resource{clusterConfig}
		if kindInstall != nil {
			dependsOnRes = append(dependsOnRes, kindInstall)
		}
		nodeImage := fmt.Sprintf("%s/%s:%s", env.InternalDockerhubMirror(), kindNodeImageName, kindVersionConfig.nodeImageVersion)
		createCluster, err := runner.Command(
			env.CommonNamer().ResourceName("kind-create-cluster"),
			&command.Args{
				Create:   pulumi.Sprintf("kind create cluster --name %s --config %s --image %s --wait %s", kindClusterName, clusterConfigFilePath, nodeImage, kindReadinessWait),
				Delete:   pulumi.Sprintf("kind delete cluster --name %s", kindClusterName),
				Triggers: pulumi.Array{pulumi.String(kindClusterConfig)},
			},
			utils.MergeOptions(opts, utils.PulumiDependsOn(dependsOnRes...), pulumi.DeleteBeforeReplace(true))...,
		)
		if err != nil {
			return err
		}

		kubeConfigCmd, err := runner.Command(
			env.CommonNamer().ResourceName("kind-kubeconfig"),
			&command.Args{
				Create: pulumi.Sprintf("kind get kubeconfig --name %s", kindClusterName),
			},
			utils.MergeOptions(opts, utils.PulumiDependsOn(createCluster))...,
		)
		if err != nil {
			return err
		}

		clusterComp.ClusterName = kindClusterName.ToStringOutput()
		if isLocal {
			clusterComp.KubeConfig = kubeConfigCmd.StdoutOutput()
			return nil
		}

		// Patch Kubeconfig based on private IP output
		// Also add skip tls
		clusterComp.KubeConfig = pulumi.All(kubeConfigCmd.StdoutOutput(), vm.Address).ApplyT(func(args []interface{}) string {
			allowInsecure := regexp.MustCompile("certificate-authority-data:.+").ReplaceAllString(args[0].(string), "insecure-skip-tls-verify: true")
			return strings.ReplaceAll(allowInsecure, "0.0.0.0", args[1].(string))
		}).(pulumi.StringOutput)

		return nil
	}, opts...)
}

func NewLocalKindCluster(env config.Env, name string, kubeVersion string, opts ...pulumi.ResourceOption) (*Cluster, error) {
	return NewKindCluster(env, nil, name, kubeVersion, opts...)
}

func runner(env config.Env, vm *remote.Host) command.Runner {
	if vm == nil {
		// local
		return command.NewLocalRunner(env, command.LocalRunnerArgs{
			// User:      user.Username,
			OSCommand: command.NewUnixOSCommand(),
		})
	}

	return vm.OS.Runner()
}

func fileManager(env config.Env, runner command.Runner) *command.FileManager {
	return command.NewFileManager(runner)
}

func installKind(env config.Env, vm *remote.Host, kindVersionConfig *kindConfig, opts ...pulumi.ResourceOption) (kindInstall command.Command, err error) {
	if vm == nil {
		return nil, nil
	}

	runner := vm.OS.Runner()
	packageManager := vm.OS.PackageManager()
	curlCommand, err := packageManager.Ensure("curl", nil, "", opts...)
	if err != nil {
		return nil, err
	}

	dockerManager, err := docker.NewManager(env, vm, opts...)
	if err != nil {
		return nil, err
	}

	opts = utils.MergeOptions(opts, utils.PulumiDependsOn(dockerManager, curlCommand))

	kindArch := vm.OS.Descriptor().Architecture
	if kindArch == os.AMD64Arch {
		kindArch = "amd64"
	}

	return runner.Command(
		env.CommonNamer().ResourceName("kind-install"),
		&command.Args{
			Create: pulumi.Sprintf(`curl --retry 10 -fsSLo ./kind "https://kind.sigs.k8s.io/dl/%s/kind-linux-%s" && sudo install kind /usr/local/bin/kind`, kindVersionConfig.kindVersion, kindArch),
		},
		opts...,
	)
}
