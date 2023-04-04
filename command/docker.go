package command

import (
	"fmt"
	"path"
	"strings"

	"github.com/DataDog/test-infra-definitions/common/namer"
	"github.com/DataDog/test-infra-definitions/common/utils"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	composeVersion = "v2.12.2"
	defaultTimeout = 300
)

type DockerComposeInlineManifest struct {
	Name    string
	Content pulumi.StringInput
}

type DockerManager struct {
	namer       namer.Namer
	runner      *Runner
	fileManager *FileManager
	pm          PackageManager
	installCmd  *remote.Command
}

func NewDockerManager(runner *Runner, packageManager PackageManager) *DockerManager {
	return &DockerManager{
		namer:       namer.NewNamer(runner.e.Ctx, "docker"),
		runner:      runner,
		fileManager: NewFileManager(runner),
		pm:          packageManager,
	}
}

func (d *DockerManager) ComposeFileUp(composeFilePath string, opts ...pulumi.ResourceOption) (*remote.Command, error) {
	installCommand, err := d.Install()
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

	opts = append(opts, utils.PulumiDependsOn(tempCmd))
	copyCmd, err := d.fileManager.CopyFile(composeFilePath, remoteComposePath, opts...)
	if err != nil {
		return nil, err
	}

	opts = append(opts, utils.PulumiDependsOn(installCommand, copyCmd))
	return d.runner.Command(
		d.namer.ResourceName("run", composeFilePath),
		&Args{
			Create: pulumi.Sprintf("docker-compose -f %s up --detach --wait --timeout %d", remoteComposePath, defaultTimeout),
			Delete: pulumi.Sprintf("docker-compose -f %s down -t %d", remoteComposePath, defaultTimeout),
			Sudo:   true,
		},
		opts...)
}

func (d *DockerManager) ComposeStrUp(name string, composeManifests []DockerComposeInlineManifest, envVars pulumi.StringMap, opts ...pulumi.ResourceOption) (*remote.Command, error) {
	installCommand, err := d.Install(opts...)
	if err != nil {
		return nil, err
	}

	tempCmd, tempDirPath, err := d.fileManager.TempDirectory(name + "compose-tmp")
	if err != nil {
		return nil, err
	}

	var remoteComposePaths []string
	runCommandTriggers := pulumi.Array{envVars}
	runCommandDeps := []pulumi.Resource{installCommand}
	for _, manifest := range composeManifests {
		remoteComposePath := path.Join(tempDirPath, fmt.Sprintf("docker-compose-%s.yml", manifest.Name))
		remoteComposePaths = append(remoteComposePaths, remoteComposePath)

		writeCommand, err := d.fileManager.CopyInlineFile(
			manifest.Content,
			remoteComposePath,
			false,
			pulumi.DependsOn([]pulumi.Resource{tempCmd}),
		)
		if err != nil {
			return nil, err
		}

		runCommandDeps = append(runCommandDeps, writeCommand)
		runCommandTriggers = append(runCommandTriggers, manifest.Content)
	}

	composeFileArgs := "-f " + strings.Join(remoteComposePaths, " -f ")

	return d.runner.Command(
		d.namer.ResourceName("compose-run", name),
		&Args{
			Create:      pulumi.Sprintf("docker-compose %s up --detach --wait --timeout %d", composeFileArgs, defaultTimeout),
			Delete:      pulumi.Sprintf("docker-compose %s down -t %d", composeFileArgs, defaultTimeout),
			Environment: envVars,
			Triggers:    runCommandTriggers,
			Sudo:        true,
		},
		pulumi.DependsOn(runCommandDeps), pulumi.DeleteBeforeReplace(true))
}

func (d *DockerManager) Install(opts ...pulumi.ResourceOption) (*remote.Command, error) {
	if d.installCmd != nil {
		return d.installCmd, nil
	}
	dockerInstall, err := d.pm.Ensure("docker.io", opts...)
	if err != nil {
		return nil, err
	}

	usermod, err := d.runner.Command(
		d.namer.ResourceName("group"),
		&Args{
			Create: pulumi.String("usermod -a -G docker $(whoami)"),
			Sudo:   true,
		},
		pulumi.DependsOn([]pulumi.Resource{dockerInstall}))
	if err != nil {
		return nil, err
	}

	composeInstallCmd := pulumi.Sprintf("curl -SL https://github.com/docker/compose/releases/download/%s/docker-compose-linux-$(uname -p) -o /usr/local/bin/docker-compose && sudo chmod 755 /usr/local/bin/docker-compose", composeVersion)
	d.installCmd, err = d.runner.Command(
		d.namer.ResourceName("install"),
		&Args{
			Create: composeInstallCmd,
			Sudo:   true,
		},
		pulumi.DependsOn([]pulumi.Resource{usermod}))
	return d.installCmd, err
}
