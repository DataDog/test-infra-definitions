package command

import (
	"path"

	"github.com/DataDog/test-infra-definitions/common/utils"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	composeVersion = "v2.12.2"
)

type DockerManager struct {
	ctx    *pulumi.Context
	runner *Runner
	pm     PackageManager
}

func NewDockerManager(ctx *pulumi.Context, runner *Runner, packageManager PackageManager) *DockerManager {
	return &DockerManager{
		ctx:    ctx,
		runner: runner,
		pm:     packageManager,
	}
}

func (d *DockerManager) ComposeFileUp(name, composeFilePath string, opts ...pulumi.ResourceOption) (*remote.Command, error) {
	installCommand, err := d.install()
	if err != nil {
		return nil, err
	}

	composeHash, err := utils.FileHash(composeFilePath)
	if err != nil {
		return nil, err
	}

	tempCmd, tempDirPath, err := TempDir(d.ctx, name+"-compose-tmp-"+composeHash, d.runner)
	if err != nil {
		return nil, err
	}
	remoteComposePath := path.Join(tempDirPath, path.Base(composeFilePath))

	composeCopy, err := remote.NewCopyFile(d.ctx, name+"-compose-copy-"+composeHash, &remote.CopyFileArgs{
		Connection: d.runner.connection,
		LocalPath:  pulumi.String(composeFilePath),
		RemotePath: pulumi.String(remoteComposePath),
	}, pulumi.DependsOn([]pulumi.Resource{tempCmd}))
	if err != nil {
		return nil, err
	}

	return d.runner.Command(d.ctx,
		name+"-compose-run-"+composeHash,
		pulumi.Sprintf("docker-compose -f %s up --detach --wait --timeout 300", remoteComposePath),
		nil,
		pulumi.Sprintf("docker-compose -f %s down -t 300", remoteComposePath),
		nil, true,
		pulumi.DependsOn([]pulumi.Resource{installCommand, composeCopy}))
}

func (d *DockerManager) ComposeStrUp(name string, composeFileContent pulumi.StringInput, opts ...pulumi.ResourceOption) (*remote.Command, error) {
	installCommand, err := d.install()
	if err != nil {
		return nil, err
	}

	tempCmd, tempDirPath, err := TempDir(d.ctx, name+"-compose-tmp", d.runner)
	if err != nil {
		return nil, err
	}
	remoteComposePath := path.Join(tempDirPath, "docker-compose.yml")

	writeCmd, err := d.runner.Command(d.ctx, name+"-compose-write", utils.WriteStringCommand(composeFileContent, remoteComposePath), nil, nil, nil, false, pulumi.DependsOn([]pulumi.Resource{tempCmd}))
	if err != nil {
		return nil, err
	}

	return d.runner.Command(d.ctx,
		name+"-compose-run",
		pulumi.Sprintf("docker-compose -f %s up --detach --wait --timeout 300", remoteComposePath),
		nil,
		pulumi.Sprintf("docker-compose -f %s down -t 300", remoteComposePath),
		nil, true,
		pulumi.DependsOn([]pulumi.Resource{installCommand, writeCmd}))
}

func (d *DockerManager) install() (*remote.Command, error) {
	dockerInstall, err := d.pm.Ensure("docker.io")
	if err != nil {
		return nil, err
	}

	usermod, err := d.runner.Command(d.ctx, "docker-group", pulumi.String("usermod -a -G docker $(whoami)"), nil, nil, nil, true, pulumi.DependsOn([]pulumi.Resource{dockerInstall}))
	if err != nil {
		return nil, err
	}

	composeInstallCmd := pulumi.Sprintf("curl -SL https://github.com/docker/compose/releases/download/%s/docker-compose-linux-$(uname -p) -o /usr/local/bin/docker-compose && sudo chmod 755 /usr/local/bin/docker-compose", composeVersion)
	return d.runner.Command(d.ctx, "docker-install", composeInstallCmd, nil, nil, nil, true, pulumi.DependsOn([]pulumi.Resource{usermod}))
}
