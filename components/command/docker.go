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

type DockerComposeManifest struct {
	Version  string                                  `yaml:"version"`
	Services map[string]DockerComposeManifestService `yaml:"services"`
}

type DockerComposeManifestService struct {
	Pid           string         `yaml:"pid,omitempty"`
	Ports         []string       `yaml:"ports,omitempty"`
	Image         string         `yaml:"image"`
	ContainerName string         `yaml:"container_name"`
	Volumes       []string       `yaml:"volumes"`
	Environment   map[string]any `yaml:"environment"`
}

type DockerComposeInlineManifest struct {
	Name    string
	Content pulumi.StringInput
}

type DockerManager struct {
	namer            namer.Namer
	runner           *Runner
	fileManager      *FileManager
	pm               PackageManager
	installCmd       *remote.Command
	ensureComposeCmd *remote.Command
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

	installComposeCommand, err := d.EnsureCompose(opts...)
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

	opts = append(opts, utils.PulumiDependsOn(installComposeCommand, copyCmd))
	return d.runner.Command(
		d.namer.ResourceName("run", composeFilePath),
		&Args{
			Create: pulumi.Sprintf("docker-compose -f %s up --detach --wait --timeout %d", remoteComposePath, defaultTimeout),
			Delete: pulumi.Sprintf("docker-compose -f %s down -t %d", remoteComposePath, defaultTimeout),
		},
		opts...)
}

func (d *DockerManager) ComposeStrUp(name string, composeManifests []DockerComposeInlineManifest, envVars pulumi.StringMap, opts ...pulumi.ResourceOption) (*remote.Command, error) {

	installComposeCommand, err := d.EnsureCompose(opts...)
	if err != nil {
		return nil, err
	}

	homeCmd, composePath, err := d.fileManager.HomeDirectory(name + "-compose-tmp")
	if err != nil {
		return nil, err
	}

	var remoteComposePaths []string
	runCommandTriggers := pulumi.Array{envVars}
	runCommandDeps := []pulumi.Resource{installComposeCommand}
	for _, manifest := range composeManifests {
		remoteComposePath := path.Join(composePath, fmt.Sprintf("docker-compose-%s.yml", manifest.Name))
		remoteComposePaths = append(remoteComposePaths, remoteComposePath)

		writeCommand, err := d.fileManager.CopyInlineFile(
			manifest.Content,
			remoteComposePath,
			false,
			pulumi.DependsOn([]pulumi.Resource{homeCmd}),
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

	whoami, err := d.runner.Command(
		d.namer.ResourceName("whoami"),
		&Args{
			Create: pulumi.String("whoami"),
			Sudo:   false,
		},
		pulumi.DependsOn([]pulumi.Resource{dockerInstall}))
	if err != nil {
		return nil, err
	}

	d.installCmd, err = d.runner.Command(
		d.namer.ResourceName("group"),
		&Args{
			Create: pulumi.Sprintf("usermod -a -G docker %s", whoami.Stdout),
			Sudo:   true,
		},
		pulumi.DependsOn([]pulumi.Resource{whoami}))
	return d.installCmd, err
}

func (d *DockerManager) EnsureCompose(opts ...pulumi.ResourceOption) (*remote.Command, error) {
	if d.ensureComposeCmd != nil {
		return d.ensureComposeCmd, nil
	}
	composeInstallIfNotCmd := pulumi.Sprintf("bash -c '(docker-compose version | grep %s) || (curl -SL https://github.com/docker/compose/releases/download/%s/docker-compose-linux-$(uname -p) -o /usr/local/bin/docker-compose && sudo chmod 755 /usr/local/bin/docker-compose)'", composeVersion, composeVersion)
	var err error
	d.ensureComposeCmd, err = d.runner.Command(
		d.namer.ResourceName("ensure-compose"),
		&Args{
			Create: composeInstallIfNotCmd,
			Sudo:   true,
		},
		opts...)
	return d.ensureComposeCmd, err
}
