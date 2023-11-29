package kubernetes

import (
	_ "embed"
	"fmt"
	"regexp"
	"strings"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/components/docker"
	"github.com/DataDog/test-infra-definitions/components/remote"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	kindReadinessWait     = "60s"
	kindNodeImageRegistry = "kindest/node"
)

//go:embed kind-cluster.yaml
var kindClusterConfig string

// Install Kind on a Linux virtual machine.
func NewKindCluster(env config.CommonEnvironment, vm *remote.Host, clusterName, kubeVersion string, opts ...pulumi.ResourceOption) (*Cluster, error) {
	return components.NewComponent(env, clusterName, func(clusterComp *Cluster) error {
		runner := vm.OS.Runner()
		commonEnvironment := env
		packageManager := vm.OS.PackageManager()
		curlCommand, err := packageManager.Ensure("curl", pulumi.Parent(clusterComp))
		if err != nil {
			return err
		}

		kindVersionConfig, err := getKindVersionConfig(kubeVersion)
		if err != nil {
			return err
		}

		dockerManager := docker.NewManager(env, vm)
		cmd, err := dockerManager.Install(pulumi.Parent(clusterComp))
		if err != nil {
			return err
		}

		kindInstall, err := runner.Command(
			commonEnvironment.CommonNamer.ResourceName("kind-install"),
			&command.Args{
				Create: pulumi.Sprintf(`curl -Lo ./kind "https://kind.sigs.k8s.io/dl/%s/kind-linux-%s" && sudo install kind /usr/local/bin/kind`, kindVersionConfig.kindVersion, vm.OS.Descriptor().Architecture),
			},
			pulumi.Parent(clusterComp),
			utils.PulumiDependsOn(cmd, curlCommand),
		)
		if err != nil {
			return err
		}

		clusterConfigFilePath := fmt.Sprintf("/tmp/kind-cluster-%s.yaml", clusterName)
		clusterConfig, err := vm.OS.FileManager().CopyInlineFile(
			pulumi.String(kindClusterConfig),
			clusterConfigFilePath, false, pulumi.Parent(clusterComp))
		if err != nil {
			return err
		}

		nodeImage := fmt.Sprintf("%s:%s", kindNodeImageRegistry, kindVersionConfig.nodeImageVersion)
		createCluster, err := runner.Command(
			commonEnvironment.CommonNamer.ResourceName("kind-create-cluster", clusterName),
			&command.Args{
				Create:   pulumi.Sprintf("kind create cluster --name %s --config %s --image %s --wait %s", clusterName, clusterConfigFilePath, nodeImage, kindReadinessWait),
				Delete:   pulumi.Sprintf("kind delete cluster --name %s", clusterName),
				Triggers: pulumi.Array{pulumi.String(kindClusterConfig)},
			},
			pulumi.Parent(clusterComp), utils.PulumiDependsOn(clusterConfig, kindInstall), pulumi.DeleteBeforeReplace(true),
		)
		if err != nil {
			return err
		}

		kubeConfigCmd, err := runner.Command(
			commonEnvironment.CommonNamer.ResourceName("kind-kubeconfig", clusterName),
			&command.Args{
				Create: pulumi.Sprintf("kind get kubeconfig --name %s", clusterName),
			},
			pulumi.Parent(clusterComp), utils.PulumiDependsOn(createCluster),
		)
		if err != nil {
			return err
		}

		// Patch Kubeconfig based on private IP output
		// Also add skip tls
		clusterComp.KubeConfig = pulumi.All(kubeConfigCmd.Stdout, vm).ApplyT(func(args []interface{}) string {
			allowInsecure := regexp.MustCompile("certificate-authority-data:.+").ReplaceAllString(args[0].(string), "insecure-skip-tls-verify: true")
			return strings.ReplaceAll(allowInsecure, "0.0.0.0", args[1].(string))
		}).(pulumi.StringOutput)

		return nil
	}, opts...)
}
