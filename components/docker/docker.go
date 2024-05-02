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
	composeVersion = "v2.27.0"
	defaultTimeout = 300
)

type Manager struct {
	namer namer.Namer
	host  *remoteComp.Host

	opts []pulumi.ResourceOption
}

func NewManager(e config.Env, host *remoteComp.Host, opts ...pulumi.ResourceOption) (*Manager, pulumi.Resource, error) {
	manager := &Manager{
		namer: e.CommonNamer().WithPrefix("docker"),
		host:  host,
		opts:  opts,
	}

	installCmd, err := manager.install()
	if err != nil {
		return nil, nil, err
	}
	manager.opts = utils.MergeOptions(manager.opts, utils.PulumiDependsOn(installCmd))

	composeCmd, err := manager.installCompose()
	if err != nil {
		return nil, nil, err
	}
	manager.opts = utils.MergeOptions(manager.opts, utils.PulumiDependsOn(composeCmd))

	return manager, composeCmd, nil
}

func (d *Manager) ComposeFileUp(composeFilePath string, opts ...pulumi.ResourceOption) (*remote.Command, error) {
	opts = utils.MergeOptions(d.opts, opts...)

	composeHash, err := utils.FileHash(composeFilePath)
	if err != nil {
		return nil, err
	}

	tempCmd, tempDirPath, err := d.host.OS.FileManager().TempDirectory(composeHash, opts...)
	if err != nil {
		return nil, err
	}
	remoteComposePath := path.Join(tempDirPath, path.Base(composeFilePath))

	copyCmd, err := d.host.OS.FileManager().CopyFile(composeFilePath, remoteComposePath, utils.MergeOptions(opts, utils.PulumiDependsOn(tempCmd))...)
	if err != nil {
		return nil, err
	}

	return d.host.OS.Runner().Command(
		d.namer.ResourceName("run", composeFilePath),
		&command.Args{
			Create: pulumi.Sprintf("docker-compose -f %s up --detach --wait --timeout %d", remoteComposePath, defaultTimeout),
			Delete: pulumi.Sprintf("docker-compose -f %s down -t %d", remoteComposePath, defaultTimeout),
		},
		utils.MergeOptions(opts, utils.PulumiDependsOn(copyCmd))...,
	)
}

func (d *Manager) ComposeStrUp(name string, composeManifests []ComposeInlineManifest, envVars pulumi.StringMap, opts ...pulumi.ResourceOption) (*remote.Command, error) {
	opts = utils.MergeOptions(d.opts, opts...)

	homeCmd, composePath, err := d.host.OS.FileManager().HomeDirectory(name+"-compose-tmp", opts...)
	if err != nil {
		return nil, err
	}
	var remoteComposePaths []string
	var manifestContents pulumi.StringArray
	runCommandDeps := make([]pulumi.Resource, 0)
	for _, manifest := range composeManifests {
		remoteComposePath := path.Join(composePath, fmt.Sprintf("docker-compose-%s.yml", manifest.Name))
		remoteComposePaths = append(remoteComposePaths, remoteComposePath)

		writeCommand, err := d.host.OS.FileManager().CopyInlineFile(
			manifest.Content,
			remoteComposePath,
			false,
			utils.MergeOptions(d.opts, utils.PulumiDependsOn(homeCmd))...,
		)
		if err != nil {
			return nil, err
		}
		manifestContents = append(manifestContents, manifest.Content)

		runCommandDeps = append(runCommandDeps, writeCommand)
	}
	contentHash := manifestContents.ToStringArrayOutput().ApplyT(func(inputs []string) string {
		mergedContent := strings.Join(inputs, "\n")
		return utils.StrHash(mergedContent)
	}).(pulumi.StringOutput)

	// We include a hash of the manifests content in the environment variables to trigger an update when a manifest changes
	// This is a workaround to avoid a force replace with Triggers when the content of the manifest changes
	envVars["CONTENT_HASH"] = contentHash

	composeFileArgs := "-f " + strings.Join(remoteComposePaths, " -f ")
	return d.host.OS.Runner().Command(
		d.namer.ResourceName("compose-run", name),
		&command.Args{
			Create:      pulumi.Sprintf("docker-compose %s up --detach --wait --timeout %d", composeFileArgs, defaultTimeout),
			Delete:      pulumi.Sprintf("docker-compose %s down -t %d", composeFileArgs, defaultTimeout),
			Environment: envVars,
		},
		utils.MergeOptions(d.opts, pulumi.DependsOn(runCommandDeps), pulumi.DeleteBeforeReplace(true))...,
	)
}

func (d *Manager) install() (*remote.Command, error) {
	dockerInstall, err := d.host.OS.PackageManager().Ensure("docker.io", nil, "docker", d.opts...)
	if err != nil {
		return nil, err
	}

	whoami, err := d.host.OS.Runner().Command(
		d.namer.ResourceName("whoami"),
		&command.Args{
			Create: pulumi.String("whoami"),
			Sudo:   false,
		},
		utils.MergeOptions(d.opts, utils.PulumiDependsOn(dockerInstall))...,
	)
	if err != nil {
		return nil, err
	}

	groupCmd, err := d.host.OS.Runner().Command(
		d.namer.ResourceName("group"),
		&command.Args{
			Create: pulumi.Sprintf("usermod -a -G docker %s", whoami.Stdout),
			Sudo:   true,
		},
		utils.MergeOptions(d.opts, utils.PulumiDependsOn(whoami))...,
	)
	if err != nil {
		return nil, err
	}

	return groupCmd, err
}

func (d *Manager) installCompose() (*remote.Command, error) {
	installCompose := pulumi.Sprintf("bash -c '(docker-compose version | grep %s) || (curl --retry 10 -fsSLo /usr/local/bin/docker-compose https://github.com/docker/compose/releases/download/%s/docker-compose-linux-$(uname -p) && sudo chmod 755 /usr/local/bin/docker-compose)'", composeVersion, composeVersion)
	return d.host.OS.Runner().Command(
		d.namer.ResourceName("install-compose"),
		&command.Args{
			Create: installCompose,
			Sudo:   true,
		},
		d.opts...)
}
