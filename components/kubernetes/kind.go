package kubernetes

import (
	_ "embed"
	"fmt"
	"regexp"
	"strings"

	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/components/vm"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	kindVersion       = "v0.18.0"
	kindReadinessWait = "60s"
)

//go:embed kind-cluster.yaml
var kindClusterConfig string

// Install Kind on a Linux virtual machine
func NewKindCluster(vm *vm.UnixVM, clusterName, arch string) (*remote.Command, pulumi.StringOutput, error) {
	runner := vm.GetRunner()
	commonEnvironment := vm.GetCommonEnvironment()
	packageManager := vm.GetPackageManager()
	curlCommand, err := packageManager.Ensure("curl")
	if err != nil {
		return nil, pulumi.StringOutput{}, err
	}
	cmd, err := vm.GetLazyDocker().Install()
	if err != nil {
		return nil, pulumi.StringOutput{}, err
	}
	kindInstall, err := runner.Command(
		commonEnvironment.CommonNamer.ResourceName("kind-install"),
		&command.Args{
			Create: pulumi.String(fmt.Sprintf(`curl -Lo ./kind "https://kind.sigs.k8s.io/dl/%s/kind-linux-%s" && sudo install kind /usr/local/bin/kind`, kindVersion, arch)),
		},
		utils.PulumiDependsOn(cmd, curlCommand),
	)
	if err != nil {
		return nil, pulumi.StringOutput{}, err
	}

	clusterConfigFilePath := fmt.Sprintf("/tmp/kind-cluster-%s.yaml", clusterName)
	fileManager := vm.GetFileManager()
	clusterConfig, err := fileManager.CopyInlineFile(
		pulumi.String(kindClusterConfig),
		clusterConfigFilePath, false)
	if err != nil {
		return nil, pulumi.StringOutput{}, err
	}

	createCluster, err := runner.Command(
		commonEnvironment.CommonNamer.ResourceName("kind-create-cluster", clusterName),
		&command.Args{
			Create:   pulumi.Sprintf("kind create cluster --name %s --config %s --wait %s", clusterName, clusterConfigFilePath, kindReadinessWait),
			Delete:   pulumi.Sprintf("sleep 10 && kind delete cluster --name %s", clusterName),
			Triggers: pulumi.Array{pulumi.String(kindClusterConfig)},
		},
		utils.PulumiDependsOn(clusterConfig, kindInstall), pulumi.DeleteBeforeReplace(true),
	)
	if err != nil {
		return nil, pulumi.StringOutput{}, err
	}

	kubeConfigCmd, err := runner.Command(
		commonEnvironment.CommonNamer.ResourceName("kind-kubeconfig", clusterName),
		&command.Args{
			Create: pulumi.Sprintf("kind get kubeconfig --name %s", clusterName),
		},
		utils.PulumiDependsOn(createCluster),
	)
	if err != nil {
		return nil, pulumi.StringOutput{}, err
	}

	// Patch Kubeconfig based on private IP output
	// Also add skip tls
	kubeConfig := pulumi.All(kubeConfigCmd.Stdout, vm.GetIP()).ApplyT(func(args []interface{}) string {
		allowInsecure := regexp.MustCompile("certificate-authority-data:.+").ReplaceAllString(args[0].(string), "insecure-skip-tls-verify: true")
		return strings.ReplaceAll(allowInsecure, "0.0.0.0", args[1].(string))
	}).(pulumi.StringOutput)

	return kubeConfigCmd, kubeConfig, nil
}
