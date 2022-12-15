package command

import (
	"path"

	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	linuxTempDir = "/tmp"
)

type FileManager struct {
	runner *Runner
}

func NewFileManager(runner *Runner) *FileManager {
	return &FileManager{
		runner: runner,
	}
}

func (fm *FileManager) CreateDirectory(name string, remotePath pulumi.StringInput, useSudo bool, opts ...pulumi.ResourceOption) (*remote.Command, error) {
	return fm.runner.Command(name,
		&CommandArgs{
			Create: pulumi.Sprintf("mkdir -p %s", remotePath),
			Delete: pulumi.Sprintf("rm -rf %s", remotePath),
			Sudo:   useSudo,
		}, opts...)
}

func (fm *FileManager) TempDirectory(name string, opts ...pulumi.ResourceOption) (*remote.Command, string, error) {
	tempDir := path.Join(linuxTempDir, name)
	folderCmd, err := fm.CreateDirectory("tmpdir-"+name, pulumi.String(tempDir), false, opts...)
	return folderCmd, tempDir, err
}

func (fm *FileManager) CopyFile(localPath, remotePath string, opts ...pulumi.ResourceOption) (*remote.CopyFile, error) {
	return remote.NewCopyFile(fm.runner.e.Ctx, fm.runner.namer.ResourceName("copy", utils.StrHash(localPath, remotePath)), &remote.CopyFileArgs{
		Connection: fm.runner.connection,
		LocalPath:  pulumi.String(localPath),
		RemotePath: pulumi.String(remotePath),
	}, opts...)
}

func (fm *FileManager) CopyInlineFile(name string, fileContent pulumi.StringInput, remotePath string, useSudo bool, opts ...pulumi.ResourceOption) (*remote.Command, error) {
	return fm.runner.Command(name,
		&CommandArgs{
			Create: utils.WriteStringCommand(remotePath),
			Stdin:  fileContent,
			Sudo:   useSudo,
		}, opts...)
}
