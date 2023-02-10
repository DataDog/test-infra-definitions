package vm

import (
	"errors"

	"github.com/DataDog/test-infra-definitions/command"
	commonos "github.com/DataDog/test-infra-definitions/common/os"
)

type UnixLikeVM struct {
	packageManager    command.PackageManager
	fileManager       *command.FileManager
	runner            *command.Runner
	lazyDockerManager *command.DockerManager
	VM
}

func NewUnixLikeVM(vm VM) (*UnixLikeVM, error) {
	os := vm.GetOS()
	if os.GetType() == commonos.WindowsType {
		return nil, errors.New("the OS Windows is not a valid Nix OS. Use `NewXXXVM` instead of `NewNixXXXVM`")
	}

	runner := vm.GetRunner()
	packageManager, err := os.CreatePackageManager(runner)
	if err != nil {
		return nil, err
	}
	return &UnixLikeVM{
		VM:             vm,
		runner:         runner,
		packageManager: packageManager,
		fileManager:    command.NewFileManager(runner),
	}, nil

}
func (vm *UnixLikeVM) GetPackageManager() command.PackageManager {
	return vm.packageManager
}

func (vm *UnixLikeVM) GetFileManager() *command.FileManager {
	return vm.fileManager
}

func (vm *UnixLikeVM) GetLazyDocker() *command.DockerManager {
	if vm.lazyDockerManager == nil {
		vm.lazyDockerManager = command.NewDockerManager(vm.GetRunner(), vm.GetPackageManager())
	}
	return vm.lazyDockerManager
}
