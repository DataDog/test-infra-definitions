package docker

import (
	"fmt"
	"path"
	"strings"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/namer"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/command"
	remoteComp "github.com/DataDog/test-infra-definitions/components/remote"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	composeVersion = "v2.12.2"
	defaultTimeout = 300
)

type ComposeInlineManifest struct {
	Name    string
	Content pulumi.StringInput
}

type Manager struct {
	namer      namer.Namer
	host       *remoteComp.Host
	installCmd *remote.Command
}

func NewManager(e config.CommonEnvironment, host *remoteComp.Host) *Manager {
	return &Manager{
		namer: e.CommonNamer.WithPrefix("docker"),
		host:  host,
	}
}

func (d *Manager) ComposeFileUp(composeFilePath string, opts ...pulumi.ResourceOption) (*remote.Command, error) {
	installComposeCommand, err := d.InstallCompose(opts...)
	if err != nil {
		return nil, err
	}

	composeHash, err := utils.FileHash(composeFilePath)
	if err != nil {
		return nil, err
	}

	tempCmd, tempDirPath, err := d.host.OS.FileManager().TempDirectory(composeHash)
	if err != nil {
		return nil, err
	}
	remoteComposePath := path.Join(tempDirPath, path.Base(composeFilePath))

	opts = utils.MergeOptions(opts, utils.PulumiDependsOn(tempCmd))
	copyCmd, err := d.host.OS.FileManager().CopyFile(composeFilePath, remoteComposePath, opts...)
	if err != nil {
		return nil, err
	}

	opts = utils.MergeOptions(opts, utils.PulumiDependsOn(installComposeCommand, copyCmd))
	return d.host.OS.Runner().Command(
		d.namer.ResourceName("run", composeFilePath),
		&command.Args{
			Create: pulumi.Sprintf("docker-compose -f %s up --detach --wait --timeout %d", remoteComposePath, defaultTimeout),
			Delete: pulumi.Sprintf("docker-compose -f %s down -t %d", remoteComposePath, defaultTimeout),
		},
		opts...)
}

func (d *Manager) ComposeStrUp(name string, composeManifests []ComposeInlineManifest, envVars pulumi.StringMap, opts ...pulumi.ResourceOption) (*remote.Command, error) {
	installComposeCommand, err := d.InstallCompose(opts...)
	if err != nil {
		return nil, err
	}

	homeCmd, composePath, err := d.host.OS.FileManager().HomeDirectory(name + "-compose-tmp")
	if err != nil {
		return nil, err
	}

	var remoteComposePaths []string
	runCommandTriggers := pulumi.Array{envVars}
	runCommandDeps := []pulumi.Resource{installComposeCommand}
	for _, manifest := range composeManifests {
		remoteComposePath := path.Join(composePath, fmt.Sprintf("docker-compose-%s.yml", manifest.Name))
		remoteComposePaths = append(remoteComposePaths, remoteComposePath)

		writeCommand, err := d.host.OS.FileManager().CopyInlineFile(
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
	return d.host.OS.Runner().Command(
		d.namer.ResourceName("compose-run", name),
		&command.Args{
			Create:      pulumi.Sprintf("docker-compose %s up --detach --wait --timeout %d", composeFileArgs, defaultTimeout),
			Delete:      pulumi.Sprintf("docker-compose %s down -t %d", composeFileArgs, defaultTimeout),
			Environment: envVars,
			Triggers:    runCommandTriggers,
		},
		pulumi.DependsOn(runCommandDeps), pulumi.DeleteBeforeReplace(true))
}

func (d *Manager) Install(opts ...pulumi.ResourceOption) (*remote.Command, error) {
	if d.installCmd != nil {
		return d.installCmd, nil
	}
	dockerInstall, err := d.host.OS.PackageManager().Ensure("docker.io", opts...)
	if err != nil {
		return nil, err
	}

	whoami, err := d.host.OS.Runner().Command(
		d.namer.ResourceName("whoami"),
		&command.Args{
			Create: pulumi.String("whoami"),
			Sudo:   false,
		},
		pulumi.DependsOn([]pulumi.Resource{dockerInstall}))
	if err != nil {
		return nil, err
	}

	d.installCmd, err = d.host.OS.Runner().Command(
		d.namer.ResourceName("group"),
		&command.Args{
			Create: pulumi.Sprintf("usermod -a -G docker %s", whoami.Stdout),
			Sudo:   true,
		},
		pulumi.DependsOn([]pulumi.Resource{whoami}))
	return d.installCmd, err
}

func (d *Manager) InstallCompose(opts ...pulumi.ResourceOption) (*remote.Command, error) {
	composeInstallIfNotCmd := pulumi.Sprintf("bash -c '(docker-compose version | grep %s) || (curl -SL https://github.com/docker/compose/releases/download/%s/docker-compose-linux-$(uname -p) -o /usr/local/bin/docker-compose && sudo chmod 755 /usr/local/bin/docker-compose)'", composeVersion, composeVersion)
	cmd, err := d.host.OS.Runner().Command(
		d.namer.ResourceName("install-compose"),
		&command.Args{
			Create: composeInstallIfNotCmd,
			Sudo:   true,
		},
		opts...)
	return cmd, err
}
