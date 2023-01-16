package kubernetes

import (
	_ "embed"
	"fmt"

	"github.com/DataDog/test-infra-definitions/command"
	"github.com/DataDog/test-infra-definitions/common/docker"
	"github.com/DataDog/test-infra-definitions/common/utils"

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
// Currently using ec2.VM waiting for correct abstraction
func NewKindCluster(dockerOnVm *docker.DockerOnVm, clusterName, arch string) (*remote.Command, error) {
	vm := dockerOnVm.GetVM()
	runner := vm.GetRunner()
	commonEnvironment := vm.GetCommonEnvironment()
	packageManager := vm.GetAptManager()
	curlCommand, err := packageManager.Ensure("curl")
	if err != nil {
		return nil, err
	}

	kindInstall, err := runner.Command(
		commonEnvironment.CommonNamer.ResourceName("kind-install"),
		&command.CommandArgs{
			Create: pulumi.Sprintf(`curl -Lo ./kind "https://kind.sigs.k8s.io/dl/%s/kind-linux-%s" && sudo install kind /usr/local/bin/kind`, kindVersion, arch),
		},
		dockerOnVm.GetDependsOn(), utils.PulumiDependsOn(curlCommand),
	)
	if err != nil {
		return nil, err
	}

	clusterConfigFilePath := fmt.Sprintf("/tmp/kind-cluster-%s.yaml", clusterName)
	fileManager := command.NewFileManager(runner)
	clusterConfig, err := fileManager.CopyInlineFile(
		commonEnvironment.CommonNamer.ResourceName("kind-cluster-config", clusterName),
		pulumi.String(kindClusterConfig),
		clusterConfigFilePath, false)
	if err != nil {
		return nil, err
	}

	createCluster, err := runner.Command(
		commonEnvironment.CommonNamer.ResourceName("kind-create-cluster", clusterName),
		&command.CommandArgs{
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
		&command.CommandArgs{
			Create: pulumi.Sprintf("kind get kubeconfig --name %s", clusterName),
		},
		utils.PulumiDependsOn(createCluster),
	)
	if err != nil {
		return nil, err
	}

	return kubeConfig, nil
}
