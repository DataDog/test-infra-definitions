package command

import (
	"fmt"
	"path"
	"strings"

	"github.com/DataDog/test-infra-definitions/common"
	"github.com/DataDog/test-infra-definitions/common/utils"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	composeVersion = "v2.12.2"
)

type DockerComposeInlineManifest struct {
	Name    string
	Content pulumi.StringInput
}

type DockerManager struct {
	namer       common.Namer
	runner      *Runner
	fileManager *FileManager
	pm          PackageManager
}

func NewDockerManager(runner *Runner, packageManager PackageManager) *DockerManager {
	return &DockerManager{
		namer:       common.NewNamer(runner.e.Ctx, "docker"),
		runner:      runner,
		fileManager: NewFileManager(runner),
		pm:          packageManager,
	}
}

func (d *DockerManager) ComposeFileUp(composeFilePath string, opts ...pulumi.ResourceOption) (*remote.Command, error) {
	installCommand, err := d.install()
	if err != nil {
		return nil, err
	}

	composeHash, err := utils.FileHash(composeFilePath)
	if err != nil {
		return nil, err
	}

	tempCmd, tempDirPath, err := d.fileManager.TempDirectory(composeHash)
	if err != nil {
		return nil, err
	}
	remoteComposePath := path.Join(tempDirPath, path.Base(composeFilePath))

	copyCmd, err := d.fileManager.CopyFile(composeFilePath, remoteComposePath, pulumi.DependsOn([]pulumi.Resource{tempCmd}))
	if err != nil {
		return nil, err
	}

	return d.runner.Command(
		d.namer.ResourceName("run", composeFilePath),
		pulumi.Sprintf("docker-compose -f %s up --detach --wait --timeout 300", remoteComposePath),
		nil,
		pulumi.Sprintf("docker-compose -f %s down -t 300", remoteComposePath),
		nil, true,
		pulumi.DependsOn([]pulumi.Resource{installCommand, copyCmd}))
}

func (d *DockerManager) ComposeStrUp(name string, composeManifests []DockerComposeInlineManifest, opts ...pulumi.ResourceOption) (*remote.Command, error) {
	installCommand, err := d.install()
	if err != nil {
		return nil, err
	}

	tempCmd, tempDirPath, err := d.fileManager.TempDirectory(name + "compose-tmp")
	if err != nil {
		return nil, err
	}

	var remoteComposePaths []string
	var writeCommands []pulumi.Resource
	for _, manifest := range composeManifests {
		remoteComposePath := path.Join(tempDirPath, fmt.Sprintf("docker-compose-%s.yml", manifest.Name))
		remoteComposePaths = append(remoteComposePaths, remoteComposePath)

		writeCommand, err := d.runner.Command(d.namer.ResourceName("write", manifest.Name), utils.WriteStringCommand(manifest.Content, remoteComposePath), nil, nil, nil, false, pulumi.DependsOn([]pulumi.Resource{tempCmd}))
		if err != nil {
			return nil, err
		}
		writeCommands = append(writeCommands, writeCommand)
	}

	composeFileArgs := "-f " + strings.Join(remoteComposePaths, " -f ")

	return d.runner.Command(
		d.namer.ResourceName("run", name),
		pulumi.Sprintf("docker-compose %s up --detach --wait --timeout 300", composeFileArgs),
		nil,
		pulumi.Sprintf("docker-compose %s down -t 300", composeFileArgs),
		nil, true,
		pulumi.DependsOn(append(writeCommands, installCommand)))
}

func (d *DockerManager) install() (*remote.Command, error) {
	dockerInstall, err := d.pm.Ensure("docker.io")
	if err != nil {
		return nil, err
	}

	usermod, err := d.runner.Command(d.namer.ResourceName("group"), pulumi.String("usermod -a -G docker $(whoami)"), nil, nil, nil, true, pulumi.DependsOn([]pulumi.Resource{dockerInstall}))
	if err != nil {
		return nil, err
	}

	composeInstallCmd := pulumi.Sprintf("curl -SL https://github.com/docker/compose/releases/download/%s/docker-compose-linux-$(uname -p) -o /usr/local/bin/docker-compose && sudo chmod 755 /usr/local/bin/docker-compose", composeVersion)
	return d.runner.Command(d.namer.ResourceName("install"), composeInstallCmd, nil, nil, nil, true, pulumi.DependsOn([]pulumi.Resource{usermod}))
}
