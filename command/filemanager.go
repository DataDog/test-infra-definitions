package command

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

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
			Create: utils.WriteStringCommand(remotePath, useSudo),
			Stdin:  fileContent,
			Sudo:   useSudo,
		}, opts...)
}

// CopyRelativeFolder copies recursively a relative folder to a remote folder.
// The path of the folder is relative to the caller of this function.
// For example, if this function is called from ~/foo/test.go with remoteFolder="bar"
// then the full path of the folder will be ~/foo/bar“.
// This function returns the resources that can be used with `pulumi.DependsOn`.
func (fm *FileManager) CopyRelativeFolder(relativeFolder string, remoteFolder string, opts ...pulumi.ResourceOption) ([]pulumi.Resource, error) {
	// `./` cannot be used with os.DirFS
	relativeFolder = strings.TrimPrefix(relativeFolder, "."+string(filepath.Separator))

	fullPath, rootFolder, err := getFullPath(relativeFolder, 2)
	if err != nil {
		return nil, err
	}

	return fm.CopyFSFolder(fullPath, os.DirFS(rootFolder), relativeFolder, remoteFolder, opts...)
}

// CopyRelativeFile copies relative path to a remote path.
// The relative path is defined in the same way as for `CopyRelativeFolder`.
// This function returns the resource that can be used with `pulumi.DependsOn`.
func (fm *FileManager) CopyRelativeFile(relativePath string, remotePath string, opts ...pulumi.ResourceOption) (pulumi.Resource, error) {
	fullPath, _, err := getFullPath(relativePath, 2)
	if err != nil {
		return nil, err
	}

	return fm.CopyFile(fullPath, remotePath, opts...)
}

// CopyFSFolder copies recursively a local folder to a remote folder.
// You may consider using `CopyRelativeFolder` which has a simpler API.
func (fm *FileManager) CopyFSFolder(
	resourceName string,
	folderFS fs.FS,
	folderPath string,
	remoteFolder string,
	opts ...pulumi.ResourceOption) ([]pulumi.Resource, error) {
	useSudo := true
	folderCommand, err := fm.CreateDirectory(resourceName, pulumi.String(remoteFolder), useSudo, opts...)
	if err != nil {
		return nil, fmt.Errorf("cannot create a temporary folder: %v for resource %v", err, resourceName)
	}

	files, folders, err := readFilesAndFolders(folderFS, folderPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read files and folders for %v. Error: %v", folderPath, err)
	}

	var folderResources []pulumi.Resource
	for _, folder := range folders {
		destFolder, err := getDestinationPath(folder, folderPath)
		if err != nil {
			return nil, err
		}
		remotePath := path.Join(remoteFolder, destFolder)
		resources, err := fm.CreateDirectory("createFolder-"+remotePath, pulumi.String(remotePath), useSudo, pulumi.DependsOn([]pulumi.Resource{folderCommand}))
		if err != nil {
			return nil, err
		}
		folderResources = append(folderResources, resources)
	}

	fileResources := []pulumi.Resource{}
	for _, file := range files {
		destFile, err := getDestinationPath(file, folderPath)

		if err != nil {
			return nil, err
		}

		fileCommand, err := fm.CopyFile(
			file,
			path.Join(remoteFolder, destFile),
			pulumi.DependsOn(folderResources))
		if err != nil {
			return nil, err
		}
		fileResources = append(fileResources, fileCommand)
	}

	return fileResources, nil
}

// When copying foo/bar to /tmp the result folder is /tmp/bar
// This function remove the root prefix from the path (`foo` in this case)
func getDestinationPath(folder string, rootFolder string) (string, error) {
	baseFolder := filepath.Base(rootFolder)
	rootWithoutBase := rootFolder[:len(rootFolder)-len(baseFolder)]

	if !strings.HasPrefix(folder, rootWithoutBase) {
		return "", fmt.Errorf("%v doesn't have the prefix %v", folder, rootWithoutBase)
	}

	return folder[len(rootWithoutBase):], nil
}

func getFullPath(relativeFolder string, skip int) (string, string, error) {
	_, file, _, ok := runtime.Caller(skip)
	if !ok {
		return "", "", errors.New("cannot get the runtime caller path")
	}
	folder := filepath.Dir(file)
	fullPath := path.Join(folder, relativeFolder)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return "", "", fmt.Errorf("the path %v doesn't exist. Caller folder: %v, relative folder: %v", fullPath, folder, relativeFolder)
	}
	return fullPath, folder, nil
}

func readFilesAndFolders(folderFS fs.FS, folderPath string) ([]string, []string, error) {
	var files []string
	var folders []string
	err := fs.WalkDir(folderFS, folderPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			folders = append(folders, path)
		} else {
			files = append(files, path)
		}
		return nil
	})

	return files, folders, err
}
