package kubernetes

import (
	_ "embed"
	"fmt"

	"github.com/DataDog/test-infra-definitions/command"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/common/vm"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	kindVersion       = "v0.17.0"
	kindReadinessWait = "60s"
)

//go:embed kind-cluster.yaml
var kindClusterConfig string

// Install Kind on a Linux virtual machine
func NewKindCluster(vm *vm.UnixVM, clusterName, arch string) (*remote.Command, error) {
	runner := vm.GetRunner()
	commonEnvironment := vm.GetCommonEnvironment()
	packageManager := vm.GetPackageManager()
	curlCommand, err := packageManager.Ensure("curl")
	if err != nil {
		return nil, err
	}
	cmd, err := vm.GetLazyDocker().Install()
	if err != nil {
		return nil, err
	}
	kindInstall, err := runner.Command(
		commonEnvironment.CommonNamer.ResourceName("kind-install"),
		&command.Args{
			Create: pulumi.Sprintf(`curl -Lo ./kind "https://kind.sigs.k8s.io/dl/%s/kind-linux-%s" && sudo install kind /usr/local/bin/kind`, kindVersion, arch),
		},
		utils.PulumiDependsOn(cmd, curlCommand),
	)
	if err != nil {
		return nil, err
	}

	clusterConfigFilePath := fmt.Sprintf("/tmp/kind-cluster-%s.yaml", clusterName)
	fileManager := vm.GetFileManager()
	clusterConfig, err := fileManager.CopyInlineFile(
		pulumi.String(kindClusterConfig),
		clusterConfigFilePath, false)
	if err != nil {
		return nil, err
	}

	createCluster, err := runner.Command(
		commonEnvironment.CommonNamer.ResourceName("kind-create-cluster", clusterName),
		&command.Args{
			Create:   pulumi.Sprintf("kind create cluster --name %s --config %s --wait %s", clusterName, clusterConfigFilePath, kindReadinessWait),
			Delete:   pulumi.Sprintf("kind delete cluster --name %s", clusterName),
			Triggers: pulumi.Array{pulumi.String(kindClusterConfig)},
		},
		utils.PulumiDependsOn(clusterConfig, kindInstall),
	)
	if err != nil {
		return nil, err
	}

	kubeConfig, err := runner.Command(
		commonEnvironment.CommonNamer.ResourceName("kind-kubeconfig", clusterName),
		&command.Args{
			Create: pulumi.Sprintf("kind get kubeconfig --name %s", clusterName),
		},
		utils.PulumiDependsOn(createCluster),
	)
	if err != nil {
		return nil, err
	}

	return kubeConfig, nil
}
